//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

// Package log implements logging with different log levels.
package log

import (
	"errors"
	"fmt"
	golog "log"
)

// Level defines logging level.
type Level int

// Logging levels.
const (
	All Level = iota
	Error
	Info
	Verbose
	Debug
	Trace
)

var (
	level = Info
)

func init() {
	golog.SetFlags(0)
}

func SetLevel(l Level) {
	level = l
}

func logf(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if len(level) > 0 {
		golog.Printf("%s: %s", level, msg)
	} else {
		golog.Printf("%s", msg)
	}
}

// Tracef logs a trace message.
func Tracef(format string, args ...interface{}) {
	if level < Trace {
		return
	}
	logf("TRACE", format, args...)
}

// Debugf logs a debugging message.
func Debugf(format string, args ...interface{}) {
	if level < Debug {
		return
	}
	logf("DEBUG", format, args...)
}

// Verbosef logs a verbose info message.
func Verbosef(format string, args ...interface{}) {
	if level < Verbose {
		return
	}
	logf("VERBOSE", format, args...)
}

// Infof logs an info message.
func Infof(format string, args ...interface{}) {
	if level < Info {
		return
	}
	logf("INFO", format, args...)
}

// Errorf logs an error message and returns the formatted error
// message as an error.
func Errorf(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	err := errors.New(msg)

	if level < Error {
		return err
	}
	logf("ERROR", "%s", msg)

	return err
}

// Fatal logs the argument and calls os.Exit(1).
func Fatal(v interface{}) {
	golog.Fatal(v)
}

// Fatalf logs the format and argument, and calls os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	golog.Fatalf(format, v...)
}

// Printf logs a log message.
func Printf(format string, args ...interface{}) {
	logf("", format, args...)
}
