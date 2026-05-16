package cmd

import (
	"fmt"
	"os"

	"gitee-cli/internal/api"
	"gitee-cli/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Global flag values populated by PersistentPreRunE.
var (
	GlobalToken    string
	GlobalHostname string
	GlobalOutput   string
	GlobalVerbose  bool
	GlobalAPIURL   string
	GlobalTimeout  int
	GlobalRepo     string
	NoColor        bool
)

var rootCmd = &cobra.Command{
	Use:   "gitee",
	Short: "Gitee CLI — manage Gitee from your terminal",
	Long: `gitee-cli is a OpenSource command-line tool for Gitee, Author: AndersHsueh
Work with repositories, issues, pull requests, gists and more.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if NoColor {
			color.NoColor = true
		}
		return nil
	},
}

// RegisterCommands adds all sub-command groups to rootCmd.
// Called from main() so that sub-packages can import cmd without cycles.
func RegisterCommands(subCmds ...*cobra.Command) {
	for _, c := range subCmds {
		rootCmd.AddCommand(c)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("Error:"), err)
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&GlobalToken, "token", "", "Gitee access token (overrides config and GITEE_TOKEN)")
	pf.StringVar(&GlobalHostname, "hostname", "gitee.com", "Gitee hostname (for enterprise deployments)")
	pf.StringVarP(&GlobalOutput, "output", "o", "table", "Output format: table|json|csv")
	pf.BoolVarP(&GlobalVerbose, "verbose", "v", false, "Show HTTP request/response details")
	pf.StringVar(&GlobalAPIURL, "api-url", "", "API base URL (defaults to https://{hostname}/api/v5)")
	pf.IntVar(&GlobalTimeout, "timeout", 30, "HTTP timeout in seconds")
	pf.StringVar(&GlobalRepo, "repo", "", "Target repository in owner/repo format")
	pf.BoolVar(&NoColor, "no-color", false, "Disable color output")
	pf.BoolP("paginate", "P", false, "Paginate through all results")
	pf.Int("page", 1, "Page number")
	pf.Int("per-page", 30, "Results per page (max 100)")
	pf.Int("limit", 0, "Maximum number of results (0 = no limit)")
}

// NewAPIClient builds an authenticated API client using the resolved token.
func NewAPIClient() (*api.Client, error) {
	token := GlobalToken
	if token == "" {
		token = config.GetToken(GlobalHostname)
	}
	if token == "" {
		return nil, fmt.Errorf("not authenticated. Set GITEE_TOKEN or run 'gitee auth login'")
	}

	baseURL := GlobalAPIURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s/api/v5", GlobalHostname)
	}

	return api.NewClient(token, baseURL, GlobalVerbose, GlobalTimeout), nil
}
