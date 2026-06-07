package help_topics

import (
	"fmt"
	"os"

	"gitee-cli/internal/output"

	"github.com/spf13/cobra"
)

// formattingTopic is the implementation of `gitee help formatting`.
//
// gitee-cli ships with three output formats. This topic explains
// the semantics, the machine-friendly patterns, and the
// --output / GITEE_OUTPUT priority rules.
var formattingTopic = &cobra.Command{
	Use:   "formatting",
	Short: "How to control output format (table/json/csv) and post-process with jq",
	Long: `Output format control for gitee-cli.

Three formats are supported today:

  table  Human-friendly columns, default for TTY output.
  json   Machine-friendly. Use for piping to jq, scripts, agents.
  csv   Spreadsheet-friendly. Header row included.

Format is selected by the --output / -o flag or GITEE_OUTPUT.
Precedence (highest first): --output > GITEE_OUTPUT > default ("table").`,
	RunE: func(c *cobra.Command, args []string) error {
		w := output.NewTableWriter()
		fmt.Fprintln(w, "FORMAT\tUSE WHEN\tNOTES")
		rows := []struct{ a, b, c string }{
			{"table", "Reading in a terminal", "Default. Auto-falls-back to plain (no color, no borders) when piped."},
			{"json", "Piping to jq / scripts / agents", "Always emits one JSON object per line for collections (NDJSON for streams), or a single object/array for singletons."},
			{"csv", "Importing to a spreadsheet", "Header row included. Quotes and embedded newlines follow RFC 4180."},
		}
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.a, r.b, r.c)
		}
		_ = w.Flush()

		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "## jq recipes")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "List recent PRs as a flat list of titles:")
		_, _ = fmt.Fprintln(os.Stdout, "  gitee pr list --output json --limit 10 | jq -r '.[].title'")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "Count open issues assigned to you:")
		_, _ = fmt.Fprintln(os.Stdout, "  gitee issue list --output json --assignee @me --state open | jq 'length'")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "Get just the URLs of the last 5 releases:")
		_, _ = fmt.Fprintln(os.Stdout, "  gitee release list --output json --limit 5 | jq -r '.[].url'")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "## Stable columns")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "gitee-cli does not currently ship a --json-fields flag; the JSON output is the")
		_, _ = fmt.Fprintln(os.Stdout, "full response from the Gitee API. To extract a subset, use jq as above. The")
		_, _ = fmt.Fprintln(os.Stdout, "field names follow Gitee API v5 (snake_case: full_name, head_ref, etc).")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "## When the API is paginated")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "Most list commands accept --paginate / -P to walk every page. Combine with")
		_, _ = fmt.Fprintln(os.Stdout, "--limit N for a total cap (N is across all pages, not per page).")
		return nil
	},
}
