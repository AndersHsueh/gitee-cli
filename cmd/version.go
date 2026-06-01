package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.4.1"

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
