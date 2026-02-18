package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mateimicu/tmux-claude-matrix/internal/logging"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// testBinary is set by TestMain to the path of a compiled test binary.
var testBinary string

func TestMain(m *testing.M) {
	// Build the binary once for subprocess tests
	tmp, err := os.MkdirTemp("", "debug-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	testBinary = filepath.Join(tmp, "claude-matrix")
	cmd := exec.Command("go", "build", "-o", testBinary, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmp)
		fmt.Fprintf(os.Stderr, "failed to build test binary: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// --- Item 8: Context helper panic tests ---

func TestConfigFromContext_PanicsWhenMissing(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		if !strings.Contains(msg, "missing config") {
			t.Errorf("panic message %q does not mention 'missing config'", msg)
		}
	}()
	configFromContext(context.Background())
}

func TestLoggerFromContext_PanicsWhenMissing(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		if !strings.Contains(msg, "missing logger") {
			t.Errorf("panic message %q does not mention 'missing logger'", msg)
		}
	}()
	loggerFromContext(context.Background())
}

func TestContextHelpers_ReturnStoredValues(t *testing.T) {
	cfg := &types.Config{Debug: true}
	log := logging.New(true)

	ctx := context.WithValue(context.Background(), configKey, cfg)
	ctx = context.WithValue(ctx, loggerKey, log)

	gotCfg := configFromContext(ctx)
	if gotCfg != cfg {
		t.Error("configFromContext returned different pointer")
	}

	gotLog := loggerFromContext(ctx)
	if gotLog != log {
		t.Error("loggerFromContext returned different pointer")
	}
}

// --- Item 6: CLI flag integration tests (subprocess) ---

func runBinary(t *testing.T, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(testBinary, args...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	// Start with a clean env, inherit PATH and HOME
	cmd.Env = append(os.Environ(), env...)

	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = -1
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestCLI_DebugFlagEnablesOutput(t *testing.T) {
	// Use diagnose because it always runs (no tmux/fzf needed) and prints
	// config info. With --debug the output should include debug mode info.
	tmpDir := t.TempDir()
	env := []string{
		"HOME=" + tmpDir,
		// Disable GitHub to avoid network calls
		"TMUX_CLAUDE_MATRIX_GITHUB_ENABLED=false",
	}

	stdout, _, _ := runBinary(t, env, "--debug", "diagnose")
	if !strings.Contains(stdout, "Debug mode: true") {
		t.Errorf("expected 'Debug mode: true' in stdout with --debug, got:\n%s", stdout)
	}
}

func TestCLI_ShortDebugFlag(t *testing.T) {
	tmpDir := t.TempDir()
	env := []string{
		"HOME=" + tmpDir,
		"TMUX_CLAUDE_MATRIX_GITHUB_ENABLED=false",
	}

	stdout, _, _ := runBinary(t, env, "-d", "diagnose")
	if !strings.Contains(stdout, "Debug mode: true") {
		t.Errorf("expected 'Debug mode: true' in stdout with -d, got:\n%s", stdout)
	}
}

func TestCLI_DefaultNoDebug(t *testing.T) {
	tmpDir := t.TempDir()
	env := []string{
		"HOME=" + tmpDir,
		"TMUX_CLAUDE_MATRIX_GITHUB_ENABLED=false",
	}

	stdout, _, _ := runBinary(t, env, "diagnose")
	if !strings.Contains(stdout, "Debug mode: false") {
		t.Errorf("expected 'Debug mode: false' in stdout without --debug, got:\n%s", stdout)
	}
}

func TestCLI_FlagOverridesEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	env := []string{
		"HOME=" + tmpDir,
		"TMUX_CLAUDE_MATRIX_DEBUG=0",
		"TMUX_CLAUDE_MATRIX_GITHUB_ENABLED=false",
	}

	// --debug should override DEBUG=0
	stdout, _, _ := runBinary(t, env, "--debug", "diagnose")
	if !strings.Contains(stdout, "Debug mode: true") {
		t.Errorf("expected --debug to override DEBUG=0, got:\n%s", stdout)
	}
}

// --- Item 7: Behavior verification tests ---

func TestCLI_DiagnoseShowsDebugMode(t *testing.T) {
	tmpDir := t.TempDir()
	env := []string{
		"HOME=" + tmpDir,
		"TMUX_CLAUDE_MATRIX_GITHUB_ENABLED=false",
	}

	// Without --debug flag
	stdout, _, _ := runBinary(t, env, "diagnose")
	if !strings.Contains(stdout, "Debug mode: false") {
		t.Errorf("diagnose without --debug should show 'Debug mode: false', got:\n%s", stdout)
	}

	// With --debug flag
	stdout2, _, _ := runBinary(t, env, "--debug", "diagnose")
	if !strings.Contains(stdout2, "Debug mode: true") {
		t.Errorf("diagnose with --debug should show 'Debug mode: true', got:\n%s", stdout2)
	}
}

func TestCLI_AuthWarningVisibleWithoutDebug(t *testing.T) {
	tmpDir := t.TempDir()
	// Create config dir
	configDir := filepath.Join(tmpDir, ".config", "tmux-claude-matrix")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Enable GitHub but provide no token — should show auth warning on stderr
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte("GITHUB_ENABLED=true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"HOME=" + tmpDir,
		// Ensure no token is available
		"GITHUB_TOKEN=",
		"GH_TOKEN=",
	}

	// Use diagnose, since create/refresh require sources to be available.
	// The diagnose command doesn't go through buildSources, but we can
	// still verify the refresh command shows the warning.
	// Use refresh which does call buildSources.
	_, stderr, _ := runBinary(t, env, "refresh")
	if !strings.Contains(stderr, "GitHub authentication not found") {
		t.Errorf("expected auth warning on stderr without debug, got stderr:\n%s", stderr)
	}
}

func TestCLI_ProgressSuppressedByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "tmux-claude-matrix")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Local source with no file — causes "no sources configured" error, but
	// before that, buildSources runs. The key is that progress messages like
	// "Refreshing repository cache..." should NOT appear on stdout.
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte("LOCAL_CONFIG_ENABLED=true\nLOCAL_REPOS_FILE="+filepath.Join(tmpDir, "repos.txt")+"\nGITHUB_ENABLED=false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create the repos file so sources are available
	if err := os.WriteFile(filepath.Join(tmpDir, "repos.txt"), []byte("https://github.com/example/repo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"HOME=" + tmpDir,
	}

	// Refresh will fail (no GitHub cache to clear, etc.) but progress messages
	// should not appear on stdout without --debug
	stdout, _, _ := runBinary(t, env, "refresh")
	if strings.Contains(stdout, "Refreshing repository cache") {
		t.Errorf("progress message should not appear on stdout without --debug, got:\n%s", stdout)
	}
}

func TestCLI_ProgressVisibleWithDebug(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "tmux-claude-matrix")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte("LOCAL_CONFIG_ENABLED=true\nLOCAL_REPOS_FILE="+filepath.Join(tmpDir, "repos.txt")+"\nGITHUB_ENABLED=false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "repos.txt"), []byte("https://github.com/example/repo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"HOME=" + tmpDir,
	}

	stdout, _, _ := runBinary(t, env, "--debug", "refresh")
	if !strings.Contains(stdout, "Refreshing repository cache") {
		t.Errorf("progress message should appear on stdout with --debug, got:\n%s", stdout)
	}
}
