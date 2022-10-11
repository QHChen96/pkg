package log

import (
	"fmt"
	"github.com/spf13/cobra"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"sort"
	"strings"
)

const (
	DefaultScopeName          = "default"
	OverrideScopeName         = "all"
	defaultOutputLevel        = InfoLevel
	defaultStackTraceLevel    = NoneLevel
	defaultOutputPath         = "stdout"
	defaultErrorOutputPath    = "stderr"
	defaultRotationMaxAge     = 30
	defaultRotationMaxSize    = 100 * 1024 * 1024
	defaultRotationMaxBackups = 1000
)

type Level int

const (
	NoneLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

var levelToString = map[Level]string{
	DebugLevel: "debug",
	InfoLevel:  "info",
	WarnLevel:  "warn",
	ErrorLevel: "error",
	FatalLevel: "fatal",
	NoneLevel:  "none",
}

var stringToLevel = map[string]Level{
	"debug": DebugLevel,
	"info":  InfoLevel,
	"warn":  WarnLevel,
	"error": ErrorLevel,
	"fatal": FatalLevel,
	"none":  NoneLevel,
}

type Options struct {
	OutputPaths        []string
	ErrorOutputPaths   []string
	RotateOutputPath   string
	RotationMaxSize    int
	RotationMaxAge     int
	RotationMaxBackups int
	JSONEncoding       bool
	LogGrpc            bool

	outputLevels     string
	logCallers       string
	stackTraceLevels string

	useStackdriverFormat     bool
	teeToStackdriver         bool
	stackdriverTargetProject string
	stackdriverQuotaProject  string
	stackdriverLogName       string
	stackdriverResource      *monitoredres.MonitoredResource

	teeToUDSServer   bool
	udsSocketAddress string
	udsServerPath    string
}

func DefaultOptions() *Options {
	return &Options{
		OutputPaths:          []string{defaultOutputPath},
		ErrorOutputPaths:     []string{defaultErrorOutputPath},
		RotationMaxSize:      defaultRotationMaxSize,
		RotationMaxAge:       defaultRotationMaxAge,
		RotationMaxBackups:   defaultRotationMaxBackups,
		outputLevels:         DefaultScopeName + ":" + levelToString[defaultOutputLevel],
		stackTraceLevels:     DefaultScopeName + ":" + levelToString[defaultStackTraceLevel],
		LogGrpc:              false,
		useStackdriverFormat: false,
	}
}

func (o *Options) WithStackdriverLoggingFormat() *Options {
	o.useStackdriverFormat = true
	return o
}

func (o *Options) WithTeeToStackdriver(project, logName string, mr *monitoredres.MonitoredResource) *Options {
	o.teeToStackdriver = true
	o.stackdriverTargetProject = project
	o.stackdriverQuotaProject = project
	o.stackdriverLogName = logName
	o.stackdriverResource = mr
	return o
}

func (o *Options) WithTeeToStackdriverWithQuotaProject(project, quotaProject, logName string, mr *monitoredres.MonitoredResource) *Options {
	o.teeToStackdriver = true
	o.stackdriverTargetProject = project
	o.stackdriverQuotaProject = quotaProject
	o.stackdriverLogName = logName
	o.stackdriverResource = mr
	return o
}

func (o *Options) WithTeeToUDS(addr, path string) *Options {
	o.teeToUDSServer = true
	o.udsSocketAddress = addr
	o.udsServerPath = path
	return o
}

func (o *Options) SetOutputLevel(scope string, level Level) {
	sl := scope + ":" + levelToString[level]
	levels := strings.Split(o.outputLevels, ",")

	if scope == DefaultScopeName {
		// see if we have an entry without a scope prefix (which represents the default scope)
		for i, ol := range levels {
			if !strings.Contains(ol, ":") {
				levels[i] = sl
				o.outputLevels = strings.Join(levels, ",")
				return
			}
		}
	}

	prefix := scope + ":"
	for i, ol := range levels {
		if strings.HasPrefix(ol, prefix) {
			levels[i] = sl
			o.outputLevels = strings.Join(levels, ",")
			return
		}
	}

	levels = append(levels, sl)
	o.outputLevels = strings.Join(levels, ",")
}

func (o *Options) GetOutputLevel(scope string) (Level, error) {
	levels := strings.Split(o.outputLevels, ",")

	if scope == DefaultScopeName {
		// see if we have an entry without a scope prefix (which represents the default scope)
		for _, ol := range levels {
			if !strings.Contains(ol, ":") {
				_, l, err := convertScopedLevel(ol)
				return l, err
			}
		}
	}

	prefix := scope + ":"
	for _, ol := range levels {
		if strings.HasPrefix(ol, prefix) {
			_, l, err := convertScopedLevel(ol)
			return l, err
		}
	}

	return NoneLevel, fmt.Errorf("no level defined for scope '%s'", scope)
}

func (o *Options) SetStackTraceLevel(scope string, level Level) {
	sl := scope + ":" + levelToString[level]
	levels := strings.Split(o.stackTraceLevels, ",")

	if scope == DefaultScopeName {
		// see if we have an entry without a scope prefix (which represents the default scope)
		for i, ol := range levels {
			if !strings.Contains(ol, ":") {
				levels[i] = sl
				o.stackTraceLevels = strings.Join(levels, ",")
				return
			}
		}
	}

	prefix := scope + ":"
	for i, ol := range levels {
		if strings.HasPrefix(ol, prefix) {
			levels[i] = sl
			o.stackTraceLevels = strings.Join(levels, ",")
			return
		}
	}

	levels = append(levels, sl)
	o.stackTraceLevels = strings.Join(levels, ",")
}

func (o *Options) GetStackTraceLevel(scope string) (Level, error) {
	levels := strings.Split(o.stackTraceLevels, ",")

	if scope == DefaultScopeName {
		// see if we have an entry without a scope prefix (which represents the default scope)
		for _, ol := range levels {
			if !strings.Contains(ol, ":") {
				_, l, err := convertScopedLevel(ol)
				return l, err
			}
		}
	}

	prefix := scope + ":"
	for _, ol := range levels {
		if strings.HasPrefix(ol, prefix) {
			_, l, err := convertScopedLevel(ol)
			return l, err
		}
	}

	return NoneLevel, fmt.Errorf("no level defined for scope '%s'", scope)
}

func (o *Options) SetLogCallers(scope string, include bool) {
	scopes := strings.Split(o.logCallers, ",")

	// remove any occurrence of the scope
	for i, s := range scopes {
		if s == scope {
			scopes[i] = ""
		}
	}

	if include {
		// find a free slot if there is one
		for i, s := range scopes {
			if s == "" {
				scopes[i] = scope
				o.logCallers = strings.Join(scopes, ",")
				return
			}
		}

		scopes = append(scopes, scope)
	}

	o.logCallers = strings.Join(scopes, ",")
}

func (o *Options) GetLogCallers(scope string) bool {
	scopes := strings.Split(o.logCallers, ",")

	for _, s := range scopes {
		if s == scope {
			return true
		}
	}

	return false
}

func convertScopedLevel(sl string) (string, Level, error) {
	var s string
	var l string

	pieces := strings.Split(sl, ":")
	if len(pieces) == 1 {
		s = DefaultScopeName
		l = pieces[0]
	} else if len(pieces) == 2 {
		s = pieces[0]
		l = pieces[1]
	} else {
		return "", NoneLevel, fmt.Errorf("invalid output level format '%s'", sl)
	}

	level, ok := stringToLevel[l]
	if !ok {
		return "", NoneLevel, fmt.Errorf("invalid output level '%s'", sl)
	}

	return s, level, nil
}

func (o *Options) AttachCobraFlags(cmd *cobra.Command) {
	o.AttachFlags(
		cmd.PersistentFlags().StringArrayVar,
		cmd.PersistentFlags().StringVar,
		cmd.PersistentFlags().IntVar,
		cmd.PersistentFlags().BoolVar)
}

func (o *Options) AttachFlags(
	stringArrayVar func(p *[]string, name string, value []string, usage string),
	stringVar func(p *string, name string, value string, usage string),
	intVar func(p *int, name string, value int, usage string),
	boolVar func(p *bool, name string, value bool, usage string),
) {
	stringArrayVar(&o.OutputPaths, "log_target", o.OutputPaths,
		"The set of paths where to output the log. This can be any path as well as the special values stdout and stderr")

	stringVar(&o.RotateOutputPath, "log_rotate", o.RotateOutputPath,
		"The path for the optional rotating log file")

	intVar(&o.RotationMaxAge, "log_rotate_max_age", o.RotationMaxAge,
		"The maximum age in days of a log file beyond which the file is rotated (0 indicates no limit)")

	intVar(&o.RotationMaxSize, "log_rotate_max_size", o.RotationMaxSize,
		"The maximum size in megabytes of a log file beyond which the file is rotated")

	intVar(&o.RotationMaxBackups, "log_rotate_max_backups", o.RotationMaxBackups,
		"The maximum number of log file backups to keep before older files are deleted (0 indicates no limit)")

	boolVar(&o.JSONEncoding, "log_as_json", o.JSONEncoding,
		"Whether to format output as JSON or in plain console-friendly format")

	levelListString := fmt.Sprintf("[%s, %s, %s, %s, %s, %s]",
		levelToString[DebugLevel],
		levelToString[InfoLevel],
		levelToString[WarnLevel],
		levelToString[ErrorLevel],
		levelToString[FatalLevel],
		levelToString[NoneLevel])

	allScopes := Scopes()
	if len(allScopes) > 1 {
		keys := make([]string, 0, len(allScopes))
		for name := range allScopes {
			keys = append(keys, name)
		}
		keys = append(keys, OverrideScopeName)
		sort.Strings(keys)
		s := strings.Join(keys, ", ")

		stringVar(&o.outputLevels, "log_output_level", o.outputLevels,
			fmt.Sprintf("Comma-separated minimum per-scope logging level of messages to output, in the form of "+
				"<scope>:<level>,<scope>:<level>,... where scope can be one of [%s] and level can be one of %s",
				s, levelListString))

		stringVar(&o.stackTraceLevels, "log_stacktrace_level", o.stackTraceLevels,
			fmt.Sprintf("Comma-separated minimum per-scope logging level at which stack traces are captured, in the form of "+
				"<scope>:<level>,<scope:level>,... where scope can be one of [%s] and level can be one of %s",
				s, levelListString))

		stringVar(&o.logCallers, "log_caller", o.logCallers,
			fmt.Sprintf("Comma-separated list of scopes for which to include caller information, scopes can be any of [%s]", s))
	} else {
		stringVar(&o.outputLevels, "log_output_level", o.outputLevels,
			fmt.Sprintf("The minimum logging level of messages to output,  can be one of %s",
				levelListString))

		stringVar(&o.stackTraceLevels, "log_stacktrace_level", o.stackTraceLevels,
			fmt.Sprintf("The minimum logging level at which stack traces are captured, can be one of %s",
				levelListString))

		stringVar(&o.logCallers, "log_caller", o.logCallers,
			"Comma-separated list of scopes for which to include called information, scopes can be any of [default]")
	}

	// NOTE: we don't currently expose a command-line option to control ErrorOutputPaths since it
	// seems too esoteric.
}
