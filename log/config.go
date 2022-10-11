// Copyright 2017 Istio Authors

package log

import (
	"go.uber.org/zap/zapcore"
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
