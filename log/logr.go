package log

import (
	"fmt"
	"github.com/go-logr/logr"
)

type zapLogger struct {
	l *Scope
}

const debugLevelThreshold = 3

func (zl *zapLogger) Enabled(level int) bool {
	if level > debugLevelThreshold {
		return zl.l.DebugEnabled()
	}
	return zl.l.InfoEnabled()
}

func trimNewline(msg string) string {
	if len(msg) == 0 {
		return msg
	}
	lc := len(msg) - 1
	if msg[lc] == '\n' {
		return msg[:lc]
	}
	return msg
}

func (zl *zapLogger) Init(logr.RuntimeInfo) {
}

func (zl *zapLogger) Info(level int, msg string, keysAndVals ...any) {
	if level > debugLevelThreshold {
		zl.l.WithLabels(keysAndVals...).Debug(trimNewline(msg))
	} else {
		zl.l.WithLabels(keysAndVals...).Info(trimNewline(msg))
	}
}

func (zl *zapLogger) Error(err error, msg string, keysAndVals ...any) {
	if zl.l.ErrorEnabled() {
		if err == nil {
			zl.l.WithLabels(keysAndVals...).Error(trimNewline(msg))
		} else {
			zl.l.WithLabels(keysAndVals...).Error(fmt.Sprintf("%v: %s", err.Error(), msg))
		}
	}
}

func (zl *zapLogger) V(int) logr.Logger {
	zlog := &zapLogger{
		l: zl.l,
	}

	return logr.New(zlog)
}

func (zl *zapLogger) WithValues(keysAndValues ...any) logr.LogSink {
	return NewLogrAdapter(zl.l.WithLabels(keysAndValues...)).GetSink()
}

func (zl *zapLogger) WithName(string) logr.LogSink {
	return zl
}

// NewLogrAdapter creates a new logr.Logger using the given Zap Logger to log.
func NewLogrAdapter(l *Scope) logr.Logger {
	zlog := &zapLogger{
		l: l,
	}

	return logr.New(zlog)
}
