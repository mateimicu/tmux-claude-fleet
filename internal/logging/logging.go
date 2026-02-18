package logging

import (
	"io"
	"os"
)

// Logger provides two io.Writer fields for debug and warning output.
// Callers use fmt.Fprintf(log.DebugW, ...) for progress messages and
// fmt.Fprintf(log.WarnW, ...) for warnings/errors.
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

// NewWithWriters creates a Logger with injectable writers for testing.
// When debug is false, debugW is ignored and DebugW is set to io.Discard.
func NewWithWriters(debug bool, debugW, warnW io.Writer) *Logger {
	if !debug {
		debugW = io.Discard
	}
	return &Logger{
		DebugW: debugW,
		WarnW:  warnW,
	}
}
