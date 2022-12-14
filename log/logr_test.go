package log

import (
	"errors"
	"github.com/go-logr/logr"
	"testing"
)

func runLogrTestWithScope(t *testing.T, s *Scope, f func(l logr.Logger)) []string {
	lines, err := captureStdout(func() {
		Configure(DefaultOptions())
		l := NewLogrAdapter(s)
		f(l)
		_ = Sync()
	})
	if err != nil {
		t.Fatalf("Got error '%v', expected success", err)
	}
	if lines[len(lines)-1] == "" {
		return lines[:len(lines)-1]
	}
	return lines
}

func newScope() *Scope {
	s := &Scope{name: "test"}
	s.SetOutputLevel(InfoLevel)
	s.SetStackTraceLevel(NoneLevel)
	s.SetLogCallers(false)
	return s
}

func runLogrTest(t *testing.T, f func(l logr.Logger)) []string {
	s := newScope()
	return runLogrTestWithScope(t, s, f)
}

func mustMatchLength(t *testing.T, l int, items []string) {
	t.Helper()
	if len(items) != l {
		t.Fatalf("expected %v items, got %v: %v", l, len(items), items)
	}
}

func TestLogr(t *testing.T) {
	t.Run("newlines not duplicated", func(t *testing.T) {
		lines := runLogrTest(t, func(l logr.Logger) {
			l.Info("msg\n")
		})
		mustMatchLength(t, 1, lines)
		mustRegexMatchString(t, lines[0], "msg")
	})
	t.Run("info level log is output", func(t *testing.T) {
		lines := runLogrTest(t, func(l logr.Logger) {
			l.Info("msg")
		})
		mustMatchLength(t, 1, lines)
		mustRegexMatchString(t, lines[0], "msg")
	})
	t.Run("error level log is output", func(t *testing.T) {
		lines := runLogrTest(t, func(l logr.Logger) {
			l.Error(errors.New("some error"), "msg")
		})
		mustMatchLength(t, 1, lines)
		mustRegexMatchString(t, lines[0], "some error.*msg")
	})
	t.Run("debug output still shows message", func(t *testing.T) {
		s := newScope()
		s.SetOutputLevel(DebugLevel)
		lines := runLogrTestWithScope(t, s, func(l logr.Logger) {
			l.Info("msg")
			l.Error(errors.New("some error"), "msg")
		})
		mustMatchLength(t, 2, lines)
		mustRegexMatchString(t, lines[0], "msg")
		mustRegexMatchString(t, lines[1], "some error.*msg")
	})
	t.Run("warn output still shows errors", func(t *testing.T) {
		s := newScope()
		s.SetOutputLevel(WarnLevel)
		lines := runLogrTestWithScope(t, s, func(l logr.Logger) {
			l.Info("msg")
			l.Error(errors.New("some error"), "msg")
		})
		mustMatchLength(t, 1, lines)
		mustRegexMatchString(t, lines[0], "some error.*msg")
	})

	t.Run("info shows correct verbosity", func(t *testing.T) {
		lines := runLogrTest(t, func(l logr.Logger) {
			l.V(0).Info("0")
			l.V(1).Info("1")
			l.V(2).Info("2")
			l.V(3).Info("3")
			l.V(4).Info("4")

			matchBool(t, true, l.V(0).Enabled())
			matchBool(t, true, l.V(3).Enabled())
			matchBool(t, false, l.V(4).Enabled())
			matchBool(t, false, l.V(6).Enabled())
		})
		mustMatchLength(t, 4, lines)
		mustRegexMatchString(t, lines[0], "0")
		mustRegexMatchString(t, lines[1], "1")
		mustRegexMatchString(t, lines[2], "2")
		mustRegexMatchString(t, lines[3], "3")
	})

	t.Run("debug shows correct verbosity", func(t *testing.T) {
		s := newScope()
		s.SetOutputLevel(DebugLevel)
		lines := runLogrTestWithScope(t, s, func(l logr.Logger) {
			l.V(0).Info("0")
			l.V(1).Info("1")
			l.V(2).Info("2")
			l.V(3).Info("3")
			l.V(4).Info("4")

			matchBool(t, true, l.V(0).Enabled())
			matchBool(t, true, l.V(3).Enabled())
			matchBool(t, true, l.V(4).Enabled())
			matchBool(t, true, l.V(6).Enabled())
		})
		mustMatchLength(t, 5, lines)
		mustRegexMatchString(t, lines[0], "0")
		mustRegexMatchString(t, lines[1], "1")
		mustRegexMatchString(t, lines[2], "2")
		mustRegexMatchString(t, lines[3], "3")
		mustRegexMatchString(t, lines[4], "4")
	})
}

func matchBool(t *testing.T, want bool, got bool) {
	t.Helper()
	if want != got {
		t.Fatalf("wanted %v got %v", want, got)
	}
}
