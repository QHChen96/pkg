package log

import (
	"cloud.google.com/go/logging"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"time"
)

var stackdriverSeverityMapping = map[zapcore.Level]logging.Severity{
	zapcore.DebugLevel:  logging.Debug,
	zapcore.InfoLevel:   logging.Info,
	zapcore.WarnLevel:   logging.Warning,
	zapcore.ErrorLevel:  logging.Error,
	zapcore.DPanicLevel: logging.Critical,
	zapcore.FatalLevel:  logging.Critical,
	zapcore.PanicLevel:  logging.Critical,
}

func encodeStackdriverLevel(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(stackdriverSeverityMapping[l].String())
}

type stackdriverCore struct {
	logger       *logging.Logger
	minimumLevel zapcore.Level
	fields       map[string]any
}

type CloseFunc func() error

func teeToStackdriver(baseCore zapcore.Core, project, quotaProject, logName string, mr *monitoredres.MonitoredResource) (zapcore.Core, CloseFunc, error) {
	if project == "" {
		return nil, func() error { return nil }, errors.New("a project must be provided for stackdriver export")
	}

	client, err := logging.NewClient(context.Background(), project, option.WithQuotaProject(quotaProject))
	if err != nil {
		return nil, func() error { return nil }, err
	}

	var logger *logging.Logger
	if mr != nil {
		logger = client.Logger(logName, logging.CommonResource(mr))
	} else {
		logger = client.Logger(logName)
	}
	sdCore := &stackdriverCore{logger: logger}

	for l := zapcore.DebugLevel; l <= zapcore.FatalLevel; l++ {
		if baseCore.Enabled(l) {
			sdCore.minimumLevel = l
			break
		}
	}

	return zapcore.NewTee(baseCore, sdCore), func() error { return client.Close() }, nil
}

func (sc *stackdriverCore) Enabled(l zapcore.Level) bool {
	return l >= sc.minimumLevel
}

func (sc *stackdriverCore) With(fields []zapcore.Field) zapcore.Core {
	return &stackdriverCore{
		logger:       sc.logger,
		minimumLevel: sc.minimumLevel,
		fields:       clone(sc.fields, fields),
	}
}

func (sc *stackdriverCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if sc.Enabled(e.Level) {
		return ce.AddCore(e, sc)
	}
	return ce
}

func (sc *stackdriverCore) Sync() error {
	if err := sc.logger.Flush(); err != nil {
		return fmt.Errorf("error writing logs to Stackdriver: %v", err)
	}
	return nil
}

func (sc *stackdriverCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	severity, specified := stackdriverSeverityMapping[entry.Level]
	if !specified {
		severity = logging.Default
	}

	payload := clone(sc.fields, fields)

	payload["logger"] = entry.LoggerName
	payload["message"] = entry.Message

	sc.logger.Log(logging.Entry{
		Timestamp: entry.Time,
		Severity:  severity,
		Payload:   payload,
	})

	return nil
}

func clone(orig map[string]any, newFields []zapcore.Field) map[string]any {
	clone := make(map[string]any)

	for k, v := range orig {
		clone[k] = v
	}

	for _, f := range newFields {
		switch f.Type {
		case zapcore.ArrayMarshalerType:
			clone[f.Key] = f.Interface
		case zapcore.ObjectMarshalerType:
			clone[f.Key] = f.Interface
		case zapcore.BinaryType:
			clone[f.Key] = f.Interface
		case zapcore.BoolType:
			clone[f.Key] = f.Integer == 1
		case zapcore.ByteStringType:
			clone[f.Key] = f.String
		case zapcore.Complex128Type:
			clone[f.Key] = fmt.Sprint(f.Interface)
		case zapcore.Complex64Type:
			clone[f.Key] = fmt.Sprint(f.Interface)
		case zapcore.DurationType:
			clone[f.Key] = time.Duration(f.Integer).String()
		case zapcore.Float64Type:
			clone[f.Key] = float64(f.Integer)
		case zapcore.Float32Type:
			clone[f.Key] = float32(f.Integer)
		case zapcore.Int64Type:
			clone[f.Key] = f.Integer
		case zapcore.Int32Type:
			clone[f.Key] = int32(f.Integer)
		case zapcore.Int16Type:
			clone[f.Key] = int16(f.Integer)
		case zapcore.Int8Type:
			clone[f.Key] = int8(f.Integer)
		case zapcore.StringType:
			clone[f.Key] = f.String
		case zapcore.TimeType:
			clone[f.Key] = f.Interface.(time.Time)
		case zapcore.Uint64Type:
			clone[f.Key] = uint64(f.Integer)
		case zapcore.Uint32Type:
			clone[f.Key] = uint32(f.Integer)
		case zapcore.Uint16Type:
			clone[f.Key] = uint16(f.Integer)
		case zapcore.Uint8Type:
			clone[f.Key] = uint8(f.Integer)
		case zapcore.UintptrType:
			clone[f.Key] = uintptr(f.Integer)
		case zapcore.ReflectType:
			clone[f.Key] = f.Interface
		case zapcore.StringerType:
			clone[f.Key] = f.Interface.(fmt.Stringer).String()
		case zapcore.ErrorType:
			clone[f.Key] = f.Interface.(error).Error()
		case zapcore.SkipType:
			continue
		default:
			clone[f.Key] = f.Interface
		}
	}

	return clone
}
