package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// installFakeExtension drops a minimal valid extension into
// <home>/.gitee-cli/extensions/<name>/ so the dynamic loader has something
// to synthesize a command from.
func installFakeExtension(t *testing.T, home, name string) {
	t.Helper()
	extDir := filepath.Join(home, ".gitee-cli", "extensions", name)
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatalf("mkdir extension: %v", err)
	}
	exe := filepath.Join(extDir, name)
	if err := os.WriteFile(exe, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write extension executable: %v", err)
	}
	manifest := `{"name":"` + name + `","path":"` + extDir + `","source":"local/test","installed_at":"2026-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(extDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write extension manifest: %v", err)
	}
}

// runWithHome runs the CLI with an explicit HOME (so the fake extension
// directory is found) plus extra env entries.
func runWithHome(t *testing.T, bin, home string, extraEnv []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = append([]string{
		"HOME=" + home,
		"PATH=/usr/bin:/bin",
		"GITEE_TOKEN=",
	}, extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestNoExtensionsFlagDisablesLoader is the regression test for the flag
// being a no-op: dynamic extension commands are loaded from main() BEFORE
// cobra parses argv, so --no-extensions could not disable the loader on its
// own. It is now pre-scanned from os.Args.
func TestNoExtensionsFlagDisablesLoader(t *testing.T) {
	bin := buildThrowawayBinary(t)
	home := t.TempDir()
	installFakeExtension(t, home, "testext")

	// Control: without the flag the extension is registered, both in the
	// auto-generated reference and in root --help's command list.
	out, err := runWithHome(t, bin, home, nil, "help", "reference")
	if err != nil {
		t.Fatalf("help reference failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "gitee testext") {
		t.Fatalf("control case: fake extension not loaded, cannot test --no-extensions\n%s", out)
	}
	helpOut, err := runWithHome(t, bin, home, nil, "--help")
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, helpOut)
	}
	if !strings.Contains(helpOut, "testext") {
		t.Fatalf("control case: extension missing from root --help\n%s", helpOut)
	}

	cases := []struct {
		name    string
		args    []string
		wantExt bool
	}{
		{"flag before command", []string{"--no-extensions", "help", "reference"}, false},
		{"flag after subcommand", []string{"help", "reference", "--no-extensions"}, false},
		{"explicitly disabled", []string{"--no-extensions=true", "help", "reference"}, false},
		{"explicitly enabled", []string{"--no-extensions=false", "help", "reference"}, true},
		{"root --help", []string{"--no-extensions", "--help"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runWithHome(t, bin, home, nil, tc.args...)
			if err != nil {
				t.Fatalf("%v failed: %v\n%s", tc.args, err, out)
			}
			got := strings.Contains(out, "gitee testext")
			if got != tc.wantExt {
				t.Errorf("%v: extension loaded = %v, want %v\n%s", tc.args, got, tc.wantExt, out)
			}
		})
	}
}

// TestNoExtensionsEnvDisablesLoader covers the env-var path through the same
// pre-parse decision (and the =0 override), so env and flag cannot drift.
func TestNoExtensionsEnvDisablesLoader(t *testing.T) {
	bin := buildThrowawayBinary(t)
	home := t.TempDir()
	installFakeExtension(t, home, "testext")

	out, err := runWithHome(t, bin, home, []string{"GITEE_NO_EXTENSIONS=1"}, "help", "reference")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if strings.Contains(out, "gitee testext") {
		t.Errorf("GITEE_NO_EXTENSIONS=1 did not disable the extension loader:\n%s", out)
	}

	out, err = runWithHome(t, bin, home, []string{"GITEE_NO_EXTENSIONS=0"}, "help", "reference")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "gitee testext") {
		t.Errorf("GITEE_NO_EXTENSIONS=0 should keep extensions loaded:\n%s", out)
	}
}

// TestExtensionsDisabledPreScan unit-tests the argv scanner directly,
// including the `--` terminator rule.
func TestExtensionsDisabledPreScan(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{"repo", "list"}, false},
		{[]string{"--no-extensions"}, true},
		{[]string{"repo", "list", "--no-extensions"}, true},
		{[]string{"--no-extensions=true"}, true},
		{[]string{"--no-extensions=1"}, true},
		{[]string{"--no-extensions=false"}, false},
		{[]string{"--no-extensions=0"}, false},
		{[]string{"--", "--no-extensions"}, false}, // after -- it is a positional arg
		{[]string{"--no-extensions", "--", "x"}, true},
	}
	for _, tc := range cases {
		t.Setenv("GITEE_NO_EXTENSIONS", "")
		if got := extensionsDisabled(tc.args); got != tc.want {
			t.Errorf("extensionsDisabled(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}

	t.Setenv("GITEE_NO_EXTENSIONS", "1")
	if !extensionsDisabled(nil) {
		t.Error("GITEE_NO_EXTENSIONS=1 should disable extensions")
	}
}
