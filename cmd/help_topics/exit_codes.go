package help_topics

import (
	"fmt"
	"os"

	"gitee-cli/internal/output"

	"github.com/spf13/cobra"
)

// exitCode describes a single exit code for the exit-codes topic table.
type exitCode struct {
	Code        string
	Meaning     string
	Origin      string
	Remediation string
}

// exitCodesTopic is the implementation of `gitee help exit-codes`.
//
// Most of gitee-cli's exit codes follow the convention:
//
//	 0  — success
//	 1  — generic / unspecified failure
//	 2  — authentication required (mirrors gh's 4 by intent; see note)
//	 3  — resource not found
//	 4  — rate limited
//	 5  — network / transport error
//	64  — usage / argument error (EX_USAGE)
//
// 130  — SIGINT (Ctrl-C)
//
// Some subcommands may emit additional codes for domain-specific
// errors (e.g. alias delete on a missing alias returns 1, not 3).
// When scripting, prefer to check for 0 / non-zero rather than
// relying on a specific code, except where documented.
var exitCodesTopic = &cobra.Command{
	Use:   "exit-codes",
	Short: "Exit codes emitted by gitee-cli commands",
	Long: `Exit codes emitted by gitee-cli.

Most users only need to distinguish 0 (success) from non-zero
(failure). When scripting against gitee-cli, prefer 0/non-zero
checks unless a specific code is documented here.`,
	RunE: func(c *cobra.Command, args []string) error {
		codes := []exitCode{
			{
				Code:        "0",
				Meaning:     "Success",
				Origin:      "all commands",
				Remediation: "n/a",
			},
			{
				Code:        "1",
				Meaning:     "Generic / unspecified failure",
				Origin:      "default; many subcommands",
				Remediation: "Re-run with --verbose / -v for the underlying error. Read the message; most are self-explanatory.",
			},
			{
				Code:        "2",
				Meaning:     "Authentication required",
				Origin:      "any command that hits the Gitee API without GITEE_TOKEN",
				Remediation: "Run 'gitee auth login' or set GITEE_TOKEN. Verify with 'gitee auth status'.",
			},
			{
				Code:        "3",
				Meaning:     "Resource not found (404)",
				Origin:      "Gitee API returned 404, or local lookup failed",
				Remediation: "Check --repo spelling, the ID/number, and that the resource is accessible to your account.",
			},
			{
				Code:        "4",
				Meaning:     "Rate limited (429)",
				Origin:      "Gitee API rate limiter; per-hour or per-minute",
				Remediation: "Wait and retry. For bulk operations, reduce --limit or use --paginate to spread the load.",
			},
			{
				Code:        "5",
				Meaning:     "Network / transport error",
				Origin:      "TLS handshake failed, DNS error, connection refused, timeout",
				Remediation: "Check connectivity, proxy, and --hostname. Re-run with --verbose for HTTP details.",
			},
			{
				Code:        "64",
				Meaning:     "Usage / argument error (EX_USAGE)",
				Origin:      "cobra: bad flags, missing positional args, validation failure",
				Remediation: "Re-run the command; the message names the missing/invalid argument.",
			},
			{
				Code:        "130",
				Meaning:     "Interrupted (SIGINT, Ctrl-C)",
				Origin:      "user",
				Remediation: "n/a — the partial operation may have left state; check with the relevant 'list' verb.",
			},
		}
		w := output.NewTableWriter()
		fmt.Fprintln(w, "CODE\tMEANING\tORIGIN\tREMEDIATION")
		for _, c := range codes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Code, c.Meaning, c.Origin, c.Remediation)
		}
		_ = w.Flush()
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "Note: gitee-cli is currently in the v0.12.x series; some subcommands may emit")
		_, _ = fmt.Fprintln(os.Stdout, "additional domain-specific codes. When scripting, prefer 0/non-zero checks")
		_, _ = fmt.Fprintln(os.Stdout, "unless you have a documented reason to depend on a specific code.")
		return nil
	},
}
