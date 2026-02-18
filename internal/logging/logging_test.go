package logging

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestLoggerWriters(t *testing.T) {
	tests := []struct {
		name       string
		debug      bool
		writeDebug string
		writeWarn  string
		wantDebug  string
		wantWarn   string
	}{
		{
			name:       "debug off discards debug output",
			debug:      false,
			writeDebug: "progress message",
			writeWarn:  "warning message",
			wantDebug:  "",
			wantWarn:   "warning message",
		},
		{
			name:       "debug off still writes warnings",
			debug:      false,
			writeDebug: "",
			writeWarn:  "⚠️  something went wrong",
			wantDebug:  "",
			wantWarn:   "⚠️  something went wrong",
		},
		{
			name:       "debug on writes debug output",
			debug:      true,
			writeDebug: "🔍 Discovering repositories...",
			writeWarn:  "⚠️  warning",
			wantDebug:  "🔍 Discovering repositories...",
			wantWarn:   "⚠️  warning",
		},
		{
			name:       "debug on writes both outputs",
			debug:      true,
			writeDebug: "progress",
			writeWarn:  "warning",
			wantDebug:  "progress",
			wantWarn:   "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var debugBuf, warnBuf bytes.Buffer
			debugW := io.Writer(&debugBuf)
			if !tt.debug {
				debugW = io.Discard
			}
			log := &Logger{DebugW: debugW, WarnW: &warnBuf}

			if tt.writeDebug != "" {
				fmt.Fprint(log.DebugW, tt.writeDebug)
			}
			if tt.writeWarn != "" {
				fmt.Fprint(log.WarnW, tt.writeWarn)
			}

			if got := debugBuf.String(); got != tt.wantDebug {
				t.Errorf("DebugW: got %q, want %q", got, tt.wantDebug)
			}
			if got := warnBuf.String(); got != tt.wantWarn {
				t.Errorf("WarnW: got %q, want %q", got, tt.wantWarn)
			}
		})
	}
}

func TestNew_DefaultWriters(t *testing.T) {
	log := New(false)
	if log.DebugW == nil {
		t.Fatal("DebugW should not be nil")
	}
	if log.WarnW == nil {
		t.Fatal("WarnW should not be nil")
	}
	// Write to DebugW should succeed (io.Discard never errors)
	n, err := fmt.Fprint(log.DebugW, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes written, got %d", n)
	}

	log2 := New(true)
	if log2.DebugW == io.Discard {
		t.Error("DebugW should not be io.Discard when debug is true")
	}
}

func TestDebugf(t *testing.T) {
	tests := []struct {
		name   string
		debug  bool
		format string
		args   []interface{}
		want   string
	}{
		{
			name:   "debug on formats message",
			debug:  true,
			format: "found %d repos\n",
			args:   []interface{}{42},
			want:   "found 42 repos\n",
		},
		{
			name:   "debug off discards message",
			debug:  false,
			format: "found %d repos\n",
			args:   []interface{}{42},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			debugW := io.Writer(&buf)
			if !tt.debug {
				debugW = io.Discard
			}
			log := &Logger{DebugW: debugW, WarnW: io.Discard}
			log.Debugf(tt.format, tt.args...)
			if got := buf.String(); got != tt.want {
				t.Errorf("Debugf: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWarnf(t *testing.T) {
	var buf bytes.Buffer
	log := &Logger{DebugW: io.Discard, WarnW: &buf}
	log.Warnf("⚠️  failed: %v\n", "timeout")
	want := "⚠️  failed: timeout\n"
	if got := buf.String(); got != want {
		t.Errorf("Warnf: got %q, want %q", got, want)
	}
}
