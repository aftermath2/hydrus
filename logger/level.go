package logger

import "errors"

// loggingLevel is the level used on each logger creation.
var loggingLevel Level

const (
	// disabled shows no logs.
	disabled Level = iota
	// fatal shows an error and exits.
	fatal
	// errorl designates error events.
	errorl
	// warning displays an alert signaling caution.
	warning
	// info designates informational messages that highlight the progress of the application at
	// coarse-grained level.
	info
	// debug designates fine-grained informational events useful to debug an application.
	debug
	// trace facilitates tracing the path of code execution within a program.
	trace
)

// Level represents the logging levels available.
type Level uint8

// LevelFromString returns the logging level corresponding to the name given.
func LevelFromString(s string) (Level, error) {
	switch s {
	case "disabled":
		return disabled, nil
	case "fatal":
		return fatal, nil
	case "error":
		return errorl, nil
	case "warning":
		return warning, nil
	case "info":
		return info, nil
	case "debug":
		return debug, nil
	case "trace":
		return trace, nil
	default:
		return 0, errors.New("logging level not found")
	}
}

// SetLoggingLevel modifies the global variable that holds the logger logging level.
func SetLoggingLevel(level Level) {
	loggingLevel = level
}

func levelName(level Level) string {
	switch level {
	case fatal:
		return "FTL"
	case errorl:
		return "ERR"
	case warning:
		return "WRN"
	case info:
		return "INF"
	case debug:
		return "DBG"
	case trace:
		return "TRC"
	default:
		return ""
	}
}
