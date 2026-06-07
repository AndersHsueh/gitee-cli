package cmd

import (
	"context"
	"fmt"
	"os"

	"gitee-cli/internal/alias"
	"gitee-cli/internal/api"
	"gitee-cli/internal/config"
	"gitee-cli/pkg/types"
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
	GlobalAgentHelp bool // --agent-help: show agent-targeted help (more verbose, machine-friendly)

	// Pagination controls (root persistent flags). Bound in init() so any
	// subcommand can read them via cmd.GlobalPaginate / GlobalPage / etc.
	GlobalPaginate bool
	GlobalPage     int
	GlobalPerPage  int
	GlobalLimit    int

	// --no-extensions: skip dynamic loader (e.g. for CI / debug). The
	// built-in `gitee extension` verb is unaffected; only the
	// dynamic commands synthesized from ~/.gitee-cli/extensions/* are
	// suppressed.
	GlobalNoExtensions bool
)

var rootCmd = &cobra.Command{
	Use:   "gitee",
	Short: "Gitee CLI — manage Gitee from your terminal",
	Long: `gitee-cli is a OpenSource command-line tool for Gitee, Author: AndersHsueh
Work with repositories, issues, pull requests, gists and more.

gitee-cli is designed for BOTH human and AI Agent use. Most "write" commands
support a separate --agent-help flag that prints extended documentation
optimized for agents: pre/post-conditions, idempotency, side effects,
common errors, and orchestration hints. Run 'gitee --agent-help' for an
overview of which commands currently have agent help.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if NoColor {
			color.NoColor = true
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		// Default behavior when no subcommand is given: show help.
		// --agent-help on the root command shows an overview of agent help coverage.
		if GlobalAgentHelp {
			fmt.Print(rootAgentHelp)
			return nil
		}
		return c.Help()
	},
}

// RegisterCommands adds all sub-command groups to rootCmd.
// Called from main() so that sub-packages can import cmd without cycles.
func RegisterCommands(subCmds ...*cobra.Command) {
	for _, c := range subCmds {
		rootCmd.AddCommand(c)
	}
}

// RegisterDynamicCommands calls each loader and attaches the returned
// *cobra.Commands to rootCmd. Call this from main() AFTER
// RegisterCommands — the loader sees the live set of built-ins and
// silently drops any extension whose name collides with one of them
// (logging a warning to stderr; the user gets the built-in).
//
// If GlobalNoExtensions is set, the loaders are NOT called at all
// (the --no-extensions flag short-circuits the whole subsystem). The
// built-in `gitee extension` verb is unaffected — it still works, it
// just doesn't see any dynamically loaded commands.
func RegisterDynamicCommands(loaders ...func() ([]*cobra.Command, error)) {
	if GlobalNoExtensions {
		return
	}
	reserved := reservedNames()
	for _, load := range loaders {
		cmds, err := load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: extension loader failed: %v\n", err)
			continue
		}
		for _, c := range cmds {
			if reserved[c.Name()] {
				fmt.Fprintf(os.Stderr, "warning: extension %q skipped (collides with built-in command)\n", c.Name())
				continue
			}
			rootCmd.AddCommand(c)
		}
	}
}

// reservedNames returns the set of command names already on rootCmd.
// Used by RegisterDynamicCommands to filter out extension name
// collisions with built-ins (we'd rather keep the built-in than let
// a user-installed extension shadow e.g. `gitee auth`).
func reservedNames() map[string]bool {
	out := make(map[string]bool, len(rootCmd.Commands()))
	for _, c := range rootCmd.Commands() {
		out[c.Name()] = true
	}
	return out
}

func Execute() {
	// Resolve user-defined aliases BEFORE cobra parses argv. This
	// is the runtime hook that makes `gitee co 123` (where `co` is
	// `gitee alias set co 'pr checkout'`) behave like
	// `gitee pr checkout 123`. Built-in commands always win, and
	// the alias store is read on every invocation so newly-added
	// aliases are picked up without restart.
	os.Args = alias.ResolveArgs(os.Args, builtInCommandNames())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("Error:"), err)
		os.Exit(1)
	}
}

// RootCommand returns the package-level rootCmd so external callers
// (main.go, integration tests) can inspect or attach commands to it
// without depending on a private symbol.
func RootCommand() *cobra.Command { return rootCmd }

// builtInCommandNames returns the set of names that are already
// registered on rootCmd at Execute() time. Used by the alias
// resolver to make built-ins win over user-defined aliases.
func builtInCommandNames() map[string]bool {
	out := make(map[string]bool, len(rootCmd.Commands()))
	for _, c := range rootCmd.Commands() {
		out[c.Name()] = true
	}
	return out
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
	pf.BoolVar(&GlobalAgentHelp, "agent-help", false, "Show agent-targeted help (long, structured, includes orchestration hints and pre/post-conditions). Available on selected commands.")
	pf.BoolVarP(&GlobalPaginate, "paginate", "P", false, "Paginate through all results (combines with --limit for total cap)")
	pf.IntVar(&GlobalPage, "page", 1, "Page number to fetch when --paginate is not set")
	pf.IntVar(&GlobalPerPage, "per-page", 30, "Results per page (max 100)")
	pf.IntVar(&GlobalLimit, "limit", 30, "Maximum number of results to return (0 = no limit). With --paginate, this is a TOTAL cap across pages.")
	pf.BoolVar(&GlobalNoExtensions, "no-extensions", false, "Skip loading external extensions (the built-in 'gitee extension' verb is unaffected)")
}

// ResolveHostname returns the effective Gitee hostname.
// Priority: --hostname flag (if explicitly set) > config active_host > "gitee.com".
// This is what makes `gitee auth switch` actually take effect — without it the
// field was write-only. (v0.10.0)
func ResolveHostname() string {
	if GlobalHostname != "gitee.com" {
		return GlobalHostname // user explicitly set --hostname to something else
	}
	if active := config.GetActiveHost(); active != "" {
		return active
	}
	return "gitee.com"
}

// NewAPIClient builds an authenticated API client using the resolved token.
func NewAPIClient() (*api.Client, error) {
	hostname := ResolveHostname()

	token := GlobalToken
	if token == "" {
		token = config.GetToken(hostname)
	}
	if token == "" {
		return nil, fmt.Errorf("not authenticated. Set GITEE_TOKEN or run 'gitee auth login'")
	}

	baseURL := GlobalAPIURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s/api/v5", hostname)
	}

	return api.NewClient(token, baseURL, GlobalVerbose, GlobalTimeout), nil
}

// ResolveCurrentUsername returns the authenticated username, trying config
// cache first then falling back to a /user API call. This is the canonical
// implementation shared by status, pr status, and issue status. (v0.10.0)
func ResolveCurrentUsername(ctx context.Context, client *api.Client) (string, error) {
	if u := config.GetUsername(GlobalHostname); u != "" {
		return u, nil
	}
	var u types.User
	if err := client.Get(ctx, "/user", &u); err != nil {
		return "", fmt.Errorf("resolving current user: %w", err)
	}
	return u.Login, nil
}
