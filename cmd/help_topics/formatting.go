package help_topics

import (
	"fmt"
	"os"

	"gitee-cli/internal/output"

	"github.com/spf13/cobra"
)

// formattingTopic is the implementation of `gitee help formatting`.
//
// gitee-cli ships with two output formats. This topic explains
// the semantics and the machine-friendly patterns. GITEE_OUTPUT
// is not read; only --output / -o selects the format.
var formattingTopic = &cobra.Command{
	Use:   "formatting",
	Short: "How to control output format (table/json) and post-process with jq",
	Long: `Output format control for gitee-cli.

Two formats are supported:

  table  Human-friendly columns. This is the default, and it is also
         what you get for any --output value other than json.
  json   One indented JSON value (an array for lists, an object for
         a single resource). Not NDJSON.

csv is not implemented. --output csv prints the table.

Format is selected only by --output / -o. GITEE_OUTPUT is not read.
Default is "table".`,
	RunE: func(c *cobra.Command, args []string) error {
		_, _ = fmt.Fprintln(os.Stdout, "csv is not implemented. --output csv prints the table.")
		_, _ = fmt.Fprintln(os.Stdout, "GITEE_OUTPUT is not read. Format is selected only by --output / -o.")
		_, _ = fmt.Fprintln(os.Stdout)
		w := output.NewTableWriter()
		fmt.Fprintln(w, "FORMAT\tUSE WHEN\tNOTES")
		rows := []struct{ a, b, c string }{
			{"table", "Reading in a terminal", "Default. Also used when --output is anything other than json, including csv."},
			{"json", "Piping to jq / scripts / agents", "One indented JSON value for the whole result. Lists are arrays, not NDJSON."},
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
		_, _ = fmt.Fprintln(os.Stdout, "Get html_url for the first page of releases (up to 5).")
		_, _ = fmt.Fprintln(os.Stdout, "JSON fields are the Go struct names from the API payload;")
		_, _ = fmt.Fprintln(os.Stdout, "a release URL is html_url, not url:")
		_, _ = fmt.Fprintln(os.Stdout, "  gitee release list --repo owner/repo --output json --limit 5 | jq -r '.[].html_url'")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "## Stable columns")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "gitee-cli does not currently ship a --json-fields flag; the JSON output is the")
		_, _ = fmt.Fprintln(os.Stdout, "full response from the Gitee API. To extract a subset, use jq as above. The")
		_, _ = fmt.Fprintln(os.Stdout, "field names follow Gitee API v5 (snake_case: full_name, head_ref, etc).")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "## When the API is paginated")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "issue list, pr list, repo list, repo commits list, status, and api")
		_, _ = fmt.Fprintln(os.Stdout, "honor --paginate / -P and walk every page. On those, --limit N is a")
		_, _ = fmt.Fprintln(os.Stdout, "total cap across pages (repo list defaults that cap to 30; pass")
		_, _ = fmt.Fprintln(os.Stdout, "--limit 0 for no cap on issue list, pr list, and repo list).")
		_, _ = fmt.Fprintln(os.Stdout, "release list, label list, milestone list, webhook list, gist list,")
		_, _ = fmt.Fprintln(os.Stdout, "ssh-key list, notification list, org lists, and user followers /")
		_, _ = fmt.Fprintln(os.Stdout, "following issue one request and ignore --paginate. For those,")
		_, _ = fmt.Fprintln(os.Stdout, "--limit is the page size of that single request.")
		return nil
	},
}
