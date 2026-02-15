package main

import (
	"bytes"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "prints set version",
			version: "v1.2.3",
			want:    "claude-matrix v1.2.3\n",
		},
		{
			name:    "prints dev when version is empty",
			version: "",
			want:    "claude-matrix dev\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			t.Cleanup(func() { Version = "" })

			var buf bytes.Buffer
			cmd := versionCmd()
			cmd.SetOut(&buf)
			cmd.SetArgs([]string{})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
