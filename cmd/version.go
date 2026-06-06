package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the semantic version of gitee-cli.
//
// Declared as `var` (not `const`) so it can be overridden at build time via:
//
//	go build -ldflags "-X gitee-cli/cmd.Version=v0.9.0-c" ./cmd/gitee
//
// This is how scripts/release.sh stamps the right version into release
// binaries without editing source code. The default value below is the
// fallback for `go build` without ldflags (e.g. local dev).
var Version = "0.9.0-c"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of gitee-cli",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gitee version %s\n", Version)
	},
}

func init() {
	rootCmd.Version = Version
	rootCmd.AddCommand(versionCmd)
	// Also support --version flag via cobra built-in
	rootCmd.SetVersionTemplate("gitee version {{.Version}}\n")
}
