package repo

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse [<owner>/<repo>]",
	Short: "Open repository in browser",
	Example: `  gitee repo browse owner/repo
  gitee repo browse --repo owner/repo`,
	RunE: runBrowse,
}

func runBrowse(cobraCmd *cobra.Command, args []string) error {
	var owner, repoName string
	var err error

	if len(args) > 0 {
		owner, repoName = splitOwnerRepo(args[0])
	} else {
		owner, repoName, err = resolveRepo("", "", nil)
		if err != nil {
			return err
		}
	}

	url := fmt.Sprintf("https://gitee.com/%s/%s", owner, repoName)

	// Try to open browser
	browser := os.Getenv("BROWSER")
	if browser == "" {
		// Try common macOS/Linux browser commands
		switch {
		case detectDarwin():
			browser = "open"
		case detectLinux():
			browser = "xdg-open"
		default:
			browser = "open"
		}
	}

	execCmd := exec.Command(browser, url)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

func detectDarwin() bool {
	return runtime.GOOS == "darwin" || exists("/usr/bin/open")
}

func detectLinux() bool {
	return runtime.GOOS == "linux" || exists("/usr/bin/xdg-open")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
