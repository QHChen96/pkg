// Copyright 2017 Istio Authors

package log

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"
	"k8s.io/klog/v2"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const (
	none          zapcore.Level = 100
	GrpcScopeName string        = "grpc"
)

var levelToZap = map[Level]zapcore.Level{
	DebugLevel: zapcore.DebugLevel,
	InfoLevel:  zapcore.InfoLevel,
	WarnLevel:  zapcore.WarnLevel,
	ErrorLevel: zapcore.ErrorLevel,
	FatalLevel: zapcore.FatalLevel,
	NoneLevel:  none,
}

var defaultEncoderConfig = zapcore.EncoderConfig{
	TimeKey:        "time",
	LevelKey:       "level",
	NameKey:        "scope",
	CallerKey:      "caller",
	MessageKey:     "msg",
	StacktraceKey:  "stack",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
	EncodeDuration: zapcore.StringDurationEncoder,
	EncodeTime:     formatDate,
}

type patchTable struct {
	write       func(ent zapcore.Entry, fields []zapcore.Field) error
	sync        func() error
	exitProcess func(code int)
	errorSink   zapcore.WriteSyncer
	close       func() error
}

var (
	// function table that can be replaced by tests
	funcs = &atomic.Value{}
	// controls whether all output is JSON or CLI style. This makes it easier to query how the zap encoder is configured
	// vs. reading it's internal state.
	useJSON atomic.Value
	logGrpc bool
)

func prepZap(options *Options) (zapcore.Core, zapcore.Core, zapcore.WriteSyncer, error) {
	var enc zapcore.Encoder
	if options.useStackdriverFormat {
		// See also: https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
		encCfg := zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "severity",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    encodeStackdriverLevel,
			EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		enc = zapcore.NewJSONEncoder(encCfg)
		useJSON.Store(true)
	} else {
		encCfg := defaultEncoderConfig

		if options.JSONEncoding {
			enc = zapcore.NewJSONEncoder(encCfg)
			useJSON.Store(true)
		} else {
			enc = zapcore.NewConsoleEncoder(encCfg)
			useJSON.Store(false)
		}
	}

	var rotaterSink zapcore.WriteSyncer
	if options.RotateOutputPath != "" {
		rotaterSink = zapcore.AddSync(&lumberjack.Logger{
			Filename:   options.RotateOutputPath,
			MaxSize:    options.RotationMaxSize,
			MaxBackups: options.RotationMaxBackups,
			MaxAge:     options.RotationMaxAge,
		})
	}

	errSink, closeErrorSink, err := zap.Open(options.ErrorOutputPaths...)
	if err != nil {
		return nil, nil, nil, err
	}

	var outputSink zapcore.WriteSyncer
	if len(options.OutputPaths) > 0 {
		outputSink, _, err = zap.Open(options.OutputPaths...)
		if err != nil {
			closeErrorSink()
			return nil, nil, nil, err
		}
	}

	var sink zapcore.WriteSyncer
	if rotaterSink != nil && outputSink != nil {
		sink = zapcore.NewMultiWriteSyncer(outputSink, rotaterSink)
	} else if rotaterSink != nil {
		sink = rotaterSink
	} else {
		sink = outputSink
	}

	var enabler zap.LevelEnablerFunc = func(lvl zapcore.Level) bool {
		switch lvl {
		case zapcore.ErrorLevel:
			return defaultScope.ErrorEnabled()
		case zapcore.WarnLevel:
			return defaultScope.WarnEnabled()
		case zapcore.InfoLevel:
			return defaultScope.InfoEnabled()
		}
		return defaultScope.DebugEnabled()
	}

	return zapcore.NewCore(enc, sink, zap.NewAtomicLevelAt(zapcore.DebugLevel)),
		zapcore.NewCore(enc, sink, enabler),
		errSink, nil
}

func formatDate(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	t = t.UTC()
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	micros := t.Nanosecond() / 1000

	buf := make([]byte, 27)

	buf[0] = byte((year/1000)%10) + '0'
	buf[1] = byte((year/100)%10) + '0'
	buf[2] = byte((year/10)%10) + '0'
	buf[3] = byte(year%10) + '0'
	buf[4] = '-'
	buf[5] = byte((month)/10) + '0'
	buf[6] = byte((month)%10) + '0'
	buf[7] = '-'
	buf[8] = byte((day)/10) + '0'
	buf[9] = byte((day)%10) + '0'
	buf[10] = 'T'
	buf[11] = byte((hour)/10) + '0'
	buf[12] = byte((hour)%10) + '0'
	buf[13] = ':'
	buf[14] = byte((minute)/10) + '0'
	buf[15] = byte((minute)%10) + '0'
	buf[16] = ':'
	buf[17] = byte((second)/10) + '0'
	buf[18] = byte((second)%10) + '0'
	buf[19] = '.'
	buf[20] = byte((micros/100000)%10) + '0'
	buf[21] = byte((micros/10000)%10) + '0'
	buf[22] = byte((micros/1000)%10) + '0'
	buf[23] = byte((micros/100)%10) + '0'
	buf[24] = byte((micros/10)%10) + '0'
	buf[25] = byte((micros)%10) + '0'
	buf[26] = 'Z'

	enc.AppendString(string(buf))
}

func updateScopes(options *Options) error {
	// snapshot what's there
	allScopes := Scopes()

	// update the output levels of all listed scopes
	if err := processLevels(allScopes, options.outputLevels, func(s *Scope, l Level) { s.SetOutputLevel(l) }); err != nil {
		return err
	}

	// update the stack tracing levels of all listed scopes
	if err := processLevels(allScopes, options.stackTraceLevels, func(s *Scope, l Level) { s.SetStackTraceLevel(l) }); err != nil {
		return err
	}

	// update the caller location setting of all listed scopes
	sc := strings.Split(options.logCallers, ",")
	for _, s := range sc {
		if s == "" {
			continue
		}

		if s == OverrideScopeName {
			// ignore everything else and just apply the override value
			for _, scope := range allScopes {
				scope.SetLogCallers(true)
			}

			return nil
		}

		if scope, ok := allScopes[s]; ok {
			scope.SetLogCallers(true)
		}
	}

	// update LogGrpc if necessary
	if logGrpc {
		options.LogGrpc = true
	}

	return nil
}

func processLevels(allScopes map[string]*Scope, arg string, setter func(*Scope, Level)) error {
	levels := strings.Split(arg, ",")
	for _, sl := range levels {
		s, l, err := convertScopedLevel(sl)
		if err != nil {
			return err
		}

		if scope, ok := allScopes[s]; ok {
			setter(scope, l)
		} else if s == OverrideScopeName {
			// override replaces everything
			for _, scope := range allScopes {
				setter(scope, l)
			}
			return nil
		} else if s == GrpcScopeName {
			grpcScope := RegisterScope(GrpcScopeName, "", 3)
			logGrpc = true
			setter(grpcScope, l)
			return nil
		}
	}

	return nil
}

func Configure(options *Options) error {
	core, captureCore, errSink, err := prepZap(options)
	if err != nil {
		return err
	}

	if err = updateScopes(options); err != nil {
		return err
	}

	closeFns := make([]func() error, 0)

	if options.teeToStackdriver {
		var closeFn, captureCloseFn func() error
		// build stackdriver core.
		core, closeFn, err = teeToStackdriver(
			core,
			options.stackdriverTargetProject,
			options.stackdriverQuotaProject,
			options.stackdriverLogName,
			options.stackdriverResource)
		if err != nil {
			return err
		}
		captureCore, captureCloseFn, err = teeToStackdriver(
			captureCore,
			options.stackdriverTargetProject,
			options.stackdriverQuotaProject,
			options.stackdriverLogName,
			options.stackdriverResource)
		if err != nil {
			return err
		}
		closeFns = append(closeFns, closeFn, captureCloseFn)
	}

	if options.teeToUDSServer {
		// build uds core.
		core = teeToUDSServer(core, options.udsSocketAddress, options.udsServerPath)
		captureCore = teeToUDSServer(captureCore, options.udsSocketAddress, options.udsServerPath)
	}

	pt := patchTable{
		write: func(ent zapcore.Entry, fields []zapcore.Field) error {
			err := core.Write(ent, fields)
			if ent.Level == zapcore.FatalLevel {
				funcs.Load().(patchTable).exitProcess(1)
			}

			return err
		},
		sync:        core.Sync,
		exitProcess: os.Exit,
		errorSink:   errSink,
		close: func() error {
			// best-effort to sync
			core.Sync() // nolint: errcheck
			for _, f := range closeFns {
				if err := f(); err != nil {
					return err
				}
			}
			return nil
		},
	}
	funcs.Store(pt)

	opts := []zap.Option{
		zap.ErrorOutput(errSink),
		zap.AddCallerSkip(1),
	}

	if defaultScope.GetLogCallers() {
		opts = append(opts, zap.AddCaller())
	}

	l := defaultScope.GetStackTraceLevel()
	if l != NoneLevel {
		opts = append(opts, zap.AddStacktrace(levelToZap[l]))
	}

	captureLogger := zap.New(captureCore, opts...)

	// capture global zap logging and force it through our logger
	_ = zap.ReplaceGlobals(captureLogger)

	// capture standard golang "log" package output and force it through our logger
	_ = zap.RedirectStdLog(captureLogger)

	// capture gRPC logging
	if options.LogGrpc {
		grpclog.SetLogger(zapgrpc.NewLogger(captureLogger.WithOptions(zap.AddCallerSkip(3))))
	}

	// capture klog (Kubernetes logging) through our logging
	configureKlog.Do(func() {
		klog.SetLogger(NewLogrAdapter(KlogScope))
	})
	// --vklog is non zero then KlogScope should be increased.
	// klog is a special case.
	if klogVerbose() {
		KlogScope.SetOutputLevel(DebugLevel)
	}

	return nil
}

func Sync() error {
	return funcs.Load().(patchTable).sync()
}

func Close() error {
	return funcs.Load().(patchTable).close()
}
