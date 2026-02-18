package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDebugConfig(t *testing.T) {
	tests := []struct {
		name       string
		configLine string // empty = no config file
		envKey     string
		envVal     string
		wantDebug  bool
	}{
		{
			name:      "default debug is false",
			wantDebug: false,
		},
		{
			name:       "config file DEBUG=1",
			configLine: "DEBUG=1",
			wantDebug:  true,
		},
		{
			name:       "config file DEBUG=true",
			configLine: "DEBUG=true",
			wantDebug:  true,
		},
		{
			name:       "config file DEBUG=0 explicit disable",
			configLine: "DEBUG=0",
			wantDebug:  false,
		},
		{
			name:       "env var overrides config file",
			configLine: "DEBUG=0",
			envKey:     "TMUX_CLAUDE_MATRIX_DEBUG",
			envVal:     "1",
			wantDebug:  true,
		},
		{
			name:      "env var alone enables debug",
			envKey:    "TMUX_CLAUDE_MATRIX_DEBUG",
			envVal:    "true",
			wantDebug: true,
		},
		{
			name:      "empty env var is ignored",
			envKey:    "TMUX_CLAUDE_MATRIX_DEBUG",
			envVal:    "",
			wantDebug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp config directory
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, ".config", "tmux-claude-matrix")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Override HOME so config.Load() finds our temp config
			t.Setenv("HOME", tmpDir)

			// Write config file if needed
			if tt.configLine != "" {
				configPath := filepath.Join(configDir, "config")
				if err := os.WriteFile(configPath, []byte(tt.configLine+"\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Set env var if needed
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}

			if cfg.Debug != tt.wantDebug {
				t.Errorf("cfg.Debug = %v, want %v", cfg.Debug, tt.wantDebug)
			}
		})
	}
}
