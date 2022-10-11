package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"khetao.com/pkg/structured"
	"runtime"
	"strings"
	"time"
)

var (
	toLevel = map[zapcore.Level]Level{
		zapcore.FatalLevel: FatalLevel,
		zapcore.ErrorLevel: ErrorLevel,
		zapcore.WarnLevel:  WarnLevel,
		zapcore.InfoLevel:  InfoLevel,
		zapcore.DebugLevel: DebugLevel,
	}
	toZapLevel = map[Level]zapcore.Level{
		FatalLevel: zapcore.FatalLevel,
		ErrorLevel: zapcore.ErrorLevel,
		WarnLevel:  zapcore.WarnLevel,
		InfoLevel:  zapcore.InfoLevel,
		DebugLevel: zapcore.DebugLevel,
	}
)

func init() {
	registerDefaultHandler(ZapLogHandlerCallbackFunc)
}

func ZapLogHandlerCallbackFunc(
	level Level,
	scope *Scope,
	ie *structured.Error,
	msg string,
) {
	var fields []zapcore.Field
	if useJSON.Load().(bool) {
		if ie != nil {
			fields = appendNotEmptyField(fields, "message", msg)
			// Unlike zap, don't leave the message in CLI format.
			msg = ""
			fields = appendNotEmptyField(fields, "moreInfo", ie.MoreInfo)
			fields = appendNotEmptyField(fields, "impact", ie.Impact)
			fields = appendNotEmptyField(fields, "action", ie.Action)
			fields = appendNotEmptyField(fields, "likelyCause", ie.LikelyCause)
			fields = appendNotEmptyField(fields, "err", toErrString(ie.Err))
		}
		for _, k := range scope.labelKeys {
			v := scope.labels[k]
			fields = append(fields, zap.Field{
				Key:       k,
				Type:      zapcore.ReflectType,
				Interface: v,
			})
		}
	} else {
		sb := &strings.Builder{}
		sb.WriteString(msg)
		if ie != nil || len(scope.labelKeys) > 0 {
			sb.WriteString("\t")
		}
		if ie != nil {
			appendNotEmptyString(sb, "moreInfo", ie.MoreInfo)
			appendNotEmptyString(sb, "impact", ie.Impact)
			appendNotEmptyString(sb, "action", ie.Action)
			appendNotEmptyString(sb, "likelyCause", ie.LikelyCause)
			appendNotEmptyString(sb, "err", toErrString(ie.Err))
		}
		space := false
		for _, k := range scope.labelKeys {
			if space {
				sb.WriteString(" ")
			}
			sb.WriteString(fmt.Sprintf("%s=%v", k, scope.labels[k]))
			space = true
		}
		msg = sb.String()
	}
	emit(scope, toZapLevel[level], msg, fields)
}

func appendNotEmptyField(fields []zapcore.Field, key, value string) []zapcore.Field {
	if key == "" || value == "" {
		return fields
	}
	return append(fields, zap.String(key, value))
}

func appendNotEmptyString(sb *strings.Builder, key, value string) {
	if key == "" || value == "" {
		return
	}
	sb.WriteString(fmt.Sprintf("%s=%v ", key, value))
}

const callerSkipOffset = 4

func dumpStack(level zapcore.Level, scope *Scope) bool {
	thresh := toLevel[level]
	if scope != defaultScope {
		thresh = ErrorLevel
		switch level {
		case zapcore.FatalLevel:
			thresh = FatalLevel
		}
	}
	return scope.GetStackTraceLevel() >= thresh
}

func emit(scope *Scope, level zapcore.Level, msg string, fields []zapcore.Field) {
	e := zapcore.Entry{
		Message:    msg,
		Level:      level,
		Time:       time.Now(),
		LoggerName: scope.nameToEmit,
	}

	if scope.GetLogCallers() {
		e.Caller = zapcore.NewEntryCaller(runtime.Caller(scope.callerSkip + callerSkipOffset))
	}

	if dumpStack(level, scope) {
		e.Stack = zap.Stack("").String
	}

	pt := funcs.Load().(patchTable)
	if pt.write != nil {
		if err := pt.write(e, fields); err != nil {
			_, _ = fmt.Fprintf(pt.errorSink, "%v log write error: %v\n", time.Now(), err)
			_ = pt.errorSink.Sync()
		}
	}
}

func toErrString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
