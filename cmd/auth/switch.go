package auth

import (
	"fmt"
	"os"

	"gitee-cli/cmd"
	"gitee-cli/internal/config"
	"gitee-cli/internal/terminal"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch [hostname]",
	Short: "Switch active Gitee host",
	Long: `Switch the active Gitee host among configured hosts.

Lists all configured hosts and lets you select one interactively.
Pass a hostname as an argument to switch directly (non-interactive).

The active host is stored in ~/.gitee-cli/settings.json and used
as the default when --hostname is not explicitly provided.`,
	Example: `  gitee auth switch
  gitee auth switch enterprise.gitee.com`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Fprintln(os.Stderr, "auth switch is an operational/config command. Use --help for usage.")
			os.Exit(0)
		}
		return nil
	},
	RunE: runSwitch,
}

func init() {
	AuthCmd.AddCommand(switchCmd)
}

func runSwitch(cobraCmd *cobra.Command, args []string) error {
	// If a hostname was passed as positional arg, switch directly.
	if len(args) >= 1 {
		target := args[0]
		hosts := config.GetAllHosts()
		found := false
		for _, h := range hosts {
			if h == target {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("hostname %q is not configured; run `gitee auth status` to see configured hosts", target)
		}
		if err := config.SetActiveHost(target); err != nil {
			return fmt.Errorf("saving active host: %w", err)
		}
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Switched active host to %s\n", green("✓"), target)
		return nil
	}

	// Interactive mode: list hosts and let user pick.
	hosts := config.GetAllHosts()
	if len(hosts) == 0 {
		fmt.Println("No hosts configured. Run `gitee auth login` first.")
		return nil // exit 0, not an error — nothing to switch to
	}

	if len(hosts) == 1 {
		// Single host: auto-select with confirmation.
		if err := config.SetActiveHost(hosts[0]); err != nil {
			return fmt.Errorf("saving active host: %w", err)
		}
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Only one host configured (%s). Set as active.\n", green("✓"), hosts[0])
		return nil
	}

	active := config.GetActiveHost()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Show current active host.
	fmt.Printf("Current active host: %s\n\n", cyan(active))

	idx, err := terminal.Select("Choose active host:", hosts)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "selection failed: %v\n", err)
		return nil // exit 0 on cancel (non-TTY, Ctrl-C, etc.)
	}

	if err := config.SetActiveHost(hosts[idx]); err != nil {
		return fmt.Errorf("saving active host: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Switched active host to %s\n", green("✓"), hosts[idx])
	return nil
}
