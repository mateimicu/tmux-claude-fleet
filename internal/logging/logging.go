package logging

import (
	"fmt"
	"io"
	"os"
)

// Logger provides two io.Writer fields for debug and warning output.
// Use the Debugf/Warnf convenience methods, or write to DebugW/WarnW
// directly when an io.Writer is needed (e.g. ghSource.SetLogger(log.DebugW)).
type Logger struct {
	DebugW io.Writer // writes only when debug enabled; io.Discard otherwise
	WarnW  io.Writer // always writes (os.Stderr)
}

// New creates a Logger with standard writers.
// When debug is true, DebugW writes to os.Stdout; otherwise io.Discard.
// WarnW always writes to os.Stderr.
func New(debug bool) *Logger {
	debugW := io.Writer(io.Discard)
	if debug {
		debugW = os.Stdout
	}
	return &Logger{
		DebugW: debugW,
		WarnW:  os.Stderr,
	}
}

// Debugf formats and writes a debug message. Output is discarded when
// debug mode is off.
func (l *Logger) Debugf(format string, args ...interface{}) {
	fmt.Fprintf(l.DebugW, format, args...) //nolint:errcheck // logging output is non-critical
}

// Warnf formats and writes a warning message. Warnings are always visible.
func (l *Logger) Warnf(format string, args ...interface{}) {
	fmt.Fprintf(l.WarnW, format, args...) //nolint:errcheck // logging output is non-critical
}
