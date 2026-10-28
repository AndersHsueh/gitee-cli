package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the semantic version of gitee-cli.
//
// Declared as `var` (not `const`) so it can be overridden at build time via:
//
//	go build -ldflags "-X gitee-cli/cmd.Version=0.13.1" ./cmd/gitee
//
// This is how scripts/build-release.sh stamps the right version into release
// binaries without editing source code. The default value below is the
// fallback for `go build` / `go install` without ldflags (e.g. local dev),
// and MUST be bumped together with README.md's version badge when cutting a
// release — TestReadmeVersionMatchesVersionVar guards that pairing.
var Version = "0.13.2"

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print the version of gitee-cli",
	Args:    cobra.NoArgs,
	PreRunE: AgentHelpNA("version"),
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
