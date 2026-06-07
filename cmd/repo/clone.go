package repo

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <owner>/<repo> [directory]",
	Short: "Clone a repository",
	Example: `  gitee repo clone owner/repo
  gitee repo clone owner/repo my-dir
  gitee repo clone owner/repo --depth 1`,
	Args: cobra.MinimumNArgs(1),
	RunE: runClone,
}

func runClone(cobraCmd *cobra.Command, args []string) error {
	repo := args[0]
	var targetDir string
	if len(args) > 1 {
		targetDir = args[1]
	} else {
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) == 2 {
			targetDir = parts[1]
		} else {
			targetDir = repo
		}
	}

	gitURL := buildGitURL(repo)

	cmdArgs := []string{"clone", gitURL}
	if targetDir != "" {
		cmdArgs = append(cmdArgs, targetDir)
	}

	execCmd := exec.Command("git", cmdArgs...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

func buildGitURL(repo string) string {
	return fmt.Sprintf("https://%s/%s", cmd.GlobalHostname, repo)
}
