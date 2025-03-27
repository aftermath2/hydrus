// Package logger contains utilities for logging relevant information.
package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

// Logger implements various functions to print information on the console.
type Logger interface {
	Trace(args ...interface{})
	Tracef(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
}

// logger contains the logging options.
type logger struct {
	out   io.Writer
	tag   string
	level Level
}

// New creates a new logger.
func New(tag string) Logger {
	return &logger{
		level: loggingLevel,
		tag:   tag,
		out:   os.Stderr,
	}
}

func (l *logger) log(level Level, message string) {
	if l.level == disabled || level > l.level {
		return
	}

	var source string
	if l.level == trace {
		_, file, line, _ := runtime.Caller(2)
		split := strings.Split(file, "/")
		join := strings.Join(split[4:], "/")
		source = fmt.Sprintf(" (%s:%d)", join, line)
	}

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05.000")
	// 2006-01-02 15:04:05.000 [TRC] API (source:1): message
	log := fmt.Sprintf("%s [%s] %s%s: %s", timestamp, levelName(level), l.tag, source, message)
	fmt.Fprintln(l.out, log)

	if l.level == fatal {
		os.Exit(1)
	}
}

// Trace provides a detailed breakdown of events for tracing the path of code execution within a program.
func (l *logger) Trace(args ...interface{}) {
	l.log(trace, fmt.Sprint(args...))
}

// Tracef is like Trace but takes a formatted message.
func (l *logger) Tracef(format string, args ...interface{}) {
	l.log(trace, fmt.Sprintf(format, args...))
}

// Debug provides useful information for debugging.
func (l *logger) Debug(args ...interface{}) {
	l.log(debug, fmt.Sprint(args...))
}

// Debugf is like Debug but takes a formatted message.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.log(debug, fmt.Sprintf(format, args...))
}

// Error reports the application errors.
func (l *logger) Error(args ...interface{}) {
	l.log(errorl, fmt.Sprint(args...))
}

// Errorf is like Error but takes a formatted message.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.log(errorl, fmt.Sprintf(format, args...))
}

// Fatal reports the application errors and exits.
func (l *logger) Fatal(args ...interface{}) {
	l.log(fatal, fmt.Sprint(args...))
}

// Fatalf is like Fatal but takes a formatted message.
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.log(fatal, fmt.Sprintf(format, args...))
}

// Info provides useful information about the server.
func (l *logger) Info(args ...interface{}) {
	l.log(info, fmt.Sprint(args...))
}

// Infof is like Info but takes a formatted message.
func (l *logger) Infof(format string, args ...interface{}) {
	l.log(info, fmt.Sprintf(format, args...))
}

// Warning reports the application alerts.
func (l *logger) Warning(args ...interface{}) {
	l.log(warning, fmt.Sprint(args...))
}

// Warningf is like Warning but takes a formatted message.
func (l *logger) Warningf(format string, args ...interface{}) {
	l.log(warning, fmt.Sprintf(format, args...))
}
