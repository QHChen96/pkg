package log

func registerDefaultScope() *Scope {
	return RegisterScope(DefaultScopeName, "Unscoped logging messages.", 1)
}

var defaultScope = registerDefaultScope()

func Fatal(fields ...any) {
	defaultScope.Fatal(fields...)
}

func Fatalf(args ...any) {
	defaultScope.Fatalf(args...)
}

func FatalEnabled() bool {
	return defaultScope.FatalEnabled()
}

func Error(fields ...any) {
	defaultScope.Error(fields...)
}

func Errorf(args ...any) {
	defaultScope.Errorf(args...)
}

func ErrorEnabled() bool {
	return defaultScope.ErrorEnabled()
}

func Warn(fields ...any) {
	defaultScope.Warn(fields...)
}

func Warna(args ...any) {
	defaultScope.Warna(args...)
}

func Warnf(args ...any) {
	defaultScope.Warnf(args...)
}

func WarnEnabled() bool {
	return defaultScope.WarnEnabled()
}

func Info(fields ...any) {
	defaultScope.Info(fields...)
}

func Infof(args ...any) {
	defaultScope.Infof(args...)
}

func InfoEnabled() bool {
	return defaultScope.InfoEnabled()
}

func Debug(fields ...any) {
	defaultScope.Debug(fields...)
}

func Debugf(args ...any) {
	defaultScope.Debugf(args...)
}

func DebugEnabled() bool {
	return defaultScope.DebugEnabled()
}

func WithLabels(kvlist ...any) *Scope {
	return defaultScope.WithLabels(kvlist...)
}
