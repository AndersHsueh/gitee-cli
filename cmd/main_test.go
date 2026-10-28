package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// sharedBinaryPath is the gitee binary built once per `go test ./cmd` run and
// shared by every test that needs to execute the real CLI.
var (
	sharedBinaryDir  string
	sharedBinaryPath string
)

// TestMain builds the CLI under test exactly once.
//
// Before this, every test that shells out (agent-help coverage/schema/matrix
// contract tests, version tests, the env integration tests, ...) called
// buildThrowawayBinary, which ran its own `go build ./gitee` into its own
// t.TempDir(). That was ~10 full builds per package run and dominated
// `go test -race ./cmd` (the binary is not raced, but every build recompiled
// the tree). One shared binary keeps the "probe the real binary" contract
// while cutting the package runtime roughly in half.
//
// The directory is created outside t.TempDir() so its lifetime spans all
// tests; TestMain removes it after m.Run().
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gitee-cli-testbin-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating shared test binary dir: %v\n", err)
		os.Exit(1)
	}
	sharedBinaryDir = dir
	sharedBinaryPath = filepath.Join(dir, "gitee")

	build := exec.Command("go", "build", "-o", sharedBinaryPath, "./gitee")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building shared test binary (`go build ./gitee`): %v\n%s", err, out)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(sharedBinaryDir)
	os.Exit(code)
}
