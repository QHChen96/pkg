package log

import (
	"fmt"
	"khetao.com/pkg/structured"
	"strings"
	"sync"
	"sync/atomic"
)

type Scope struct {
	name        string
	nameToEmit  string
	description string
	callerSkip  int

	outputLevel     atomic.Value
	stackTraceLevel atomic.Value
	logCallers      atomic.Value

	labelKeys []string
	labels    map[string]any
}

var (
	scopes = make(map[string]*Scope)
	lock   sync.RWMutex

	defaultHandlers []scopeHandlerCallbackFunc
	// Write lock should only be taken during program startup.
	defaultHandlersMu sync.RWMutex
)

type scopeHandlerCallbackFunc func(
	level Level,
	scope *Scope,
	ie *structured.Error,
	msg string)

func registerDefaultHandler(callback scopeHandlerCallbackFunc) {
	defaultHandlersMu.Lock()
	defer defaultHandlersMu.Unlock()
	defaultHandlers = append(defaultHandlers, callback)
}

func RegisterScope(name string, description string, callerSkip int) *Scope {
	if strings.ContainsAny(name, ":,.") {
		panic(fmt.Sprintf("scope name %s is invalid, it cannot contain colons, commas, or periods", name))
	}

	lock.Lock()
	defer lock.Unlock()

	s, ok := scopes[name]
	if !ok {
		s = &Scope{
			name:        name,
			description: description,
			callerSkip:  callerSkip,
		}
		s.SetOutputLevel(InfoLevel)
		s.SetStackTraceLevel(NoneLevel)
		s.SetLogCallers(false)

		if name != DefaultScopeName {
			s.nameToEmit = name
		}

		scopes[name] = s
	}

	s.labels = make(map[string]any)

	return s
}

func FindScope(scope string) *Scope {
	lock.RLock()
	defer lock.RUnlock()

	s := scopes[scope]
	return s
}

func Scopes() map[string]*Scope {
	lock.RLock()
	defer lock.RUnlock()

	s := make(map[string]*Scope, len(scopes))
	for k, v := range scopes {
		s[k] = v
	}

	return s
}

func (s *Scope) Fatal(args ...any) {
	if s.GetOutputLevel() >= FatalLevel {
		ie, firstIdx := getErrorStruct(args)
		if firstIdx == 0 {
			s.callHandlers(FatalLevel, s, ie, fmt.Sprint(args...))
			return
		}
		s.callHandlers(FatalLevel, s, ie, fmt.Sprint(args[firstIdx:]...))
	}
}

func (s *Scope) Fatalf(args ...any) {
	if s.GetOutputLevel() >= FatalLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(FatalLevel, s, ie, msg)
	}
}

func (s *Scope) FatalEnabled() bool {
	return s.GetOutputLevel() >= FatalLevel
}

func (s *Scope) Error(args ...any) {
	if s.GetOutputLevel() >= ErrorLevel {
		ie, firstIdx := getErrorStruct(args)
		if firstIdx == 0 {
			s.callHandlers(ErrorLevel, s, ie, fmt.Sprint(args...))
			return
		}
		s.callHandlers(ErrorLevel, s, ie, fmt.Sprint(args[firstIdx:]...))
	}
}

func (s *Scope) Errorf(args ...any) {
	if s.GetOutputLevel() >= ErrorLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(ErrorLevel, s, ie, msg)
	}
}

func (s *Scope) ErrorEnabled() bool {
	return s.GetOutputLevel() >= ErrorLevel
}

func (s *Scope) Warn(args ...any) {
	if s.GetOutputLevel() >= WarnLevel {
		ie, firstIdx := getErrorStruct(args)
		if firstIdx == 0 {
			s.callHandlers(WarnLevel, s, ie, fmt.Sprint(args...))
			return
		}
		s.callHandlers(WarnLevel, s, ie, fmt.Sprint(args[firstIdx:]...))
	}
}

func (s *Scope) Warna(args ...any) {
	s.Warn(args...)
}

func (s *Scope) Warnf(args ...any) {
	if s.GetOutputLevel() >= WarnLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(WarnLevel, s, ie, msg)
	}
}

func (s *Scope) WarnEnabled() bool {
	return s.GetOutputLevel() >= WarnLevel
}

func (s *Scope) Info(args ...any) {
	if s.GetOutputLevel() >= InfoLevel {
		ie, firstIdx := getErrorStruct(args)
		if firstIdx == 0 {
			s.callHandlers(InfoLevel, s, ie, fmt.Sprint(args...))
			return
		}
		s.callHandlers(InfoLevel, s, ie, fmt.Sprint(args[firstIdx:]...))
	}
}

func (s *Scope) Infof(args ...any) {
	if s.GetOutputLevel() >= InfoLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(InfoLevel, s, ie, msg)
	}
}

func (s *Scope) InfoEnabled() bool {
	return s.GetOutputLevel() >= InfoLevel
}

func (s *Scope) Debug(args ...any) {
	if s.GetOutputLevel() >= DebugLevel {
		ie, firstIdx := getErrorStruct(args)
		if firstIdx == 0 {
			s.callHandlers(DebugLevel, s, ie, fmt.Sprint(args...))
			return
		}
		s.callHandlers(DebugLevel, s, ie, fmt.Sprint(args[firstIdx:]...))
	}
}

func (s *Scope) Debugf(args ...any) {
	if s.GetOutputLevel() >= DebugLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(DebugLevel, s, ie, msg)
	}
}

func (s *Scope) DebugEnabled() bool {
	return s.GetOutputLevel() >= DebugLevel
}

func (s *Scope) Name() string {
	return s.name
}

func (s *Scope) Description() string {
	return s.description
}

func (s *Scope) SetOutputLevel(l Level) {
	s.outputLevel.Store(l)
}

func (s *Scope) GetOutputLevel() Level {
	return s.outputLevel.Load().(Level)
}

func (s *Scope) SetStackTraceLevel(l Level) {
	s.stackTraceLevel.Store(l)
}

func (s *Scope) GetStackTraceLevel() Level {
	return s.stackTraceLevel.Load().(Level)
}

func (s *Scope) SetLogCallers(logCallers bool) {
	s.logCallers.Store(logCallers)
}

func (s *Scope) GetLogCallers() bool {
	return s.logCallers.Load().(bool)
}

func (s *Scope) copy() *Scope {
	out := *s
	out.labels = copyStringInterfaceMap(s.labels)
	return &out
}

func (s *Scope) WithLabels(kvlist ...any) *Scope {
	out := s.copy()
	if len(kvlist)%2 != 0 {
		out.labels["WithLabels error"] = fmt.Sprintf("even number of parameters required, got %d", len(kvlist))
		return out
	}

	for i := 0; i < len(kvlist); i += 2 {
		keyi := kvlist[i]
		key, ok := keyi.(string)
		if !ok {
			out.labels["WithLabels error"] = fmt.Sprintf("label name %v must be a string, got %T ", keyi, keyi)
			return out
		}
		out.labels[key] = kvlist[i+1]
		out.labelKeys = append(out.labelKeys, key)
	}
	return out
}

func (s *Scope) callHandlers(
	severity Level,
	scope *Scope,
	ie *structured.Error,
	msg string,
) {
	defaultHandlersMu.RLock()
	defer defaultHandlersMu.RUnlock()
	for _, h := range defaultHandlers {
		h(severity, scope, ie, msg)
	}
}

func getErrorStruct(fields ...any) (*structured.Error, int) {
	ief, ok := fields[0].([]any)
	if !ok {
		return nil, 0
	}
	ie, ok := ief[0].(*structured.Error)
	if !ok {
		return nil, 0
	}
	// Skip Error, pass remaining fields on as before.
	return ie, 1
}

func copyStringInterfaceMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
