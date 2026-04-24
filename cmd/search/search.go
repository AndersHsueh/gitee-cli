package search

import (
"context"
"encoding/json"
"fmt"
"os"
"text/tabwriter"
"time"

"github.com/spf13/cobra"
"gitee-cli/cmd"
"gitee-cli/internal/api"
"gitee-cli/internal/config"
)

var SearchCmd = &cobra.Command{
Use:   "search <subcommand>",
Short: "Search for repositories, issues, or users on Gitee",
}

// newSearchClient creates a client; token is optional for public search endpoints
func newSearchClient() *api.Client {
token := cmd.GlobalToken
if token == "" {
token = config.GetToken(cmd.GlobalHostname)
}
return api.NewClient(token, cmd.GlobalAPIURL, cmd.GlobalVerbose, cmd.GlobalTimeout)
}

// ----- repos -----

var reposFlags struct {
language string
sort     string
order    string
limit    int
}

var searchReposCmd = &cobra.Command{
Use:   "repos <query>",
Short: "Search repositories",
Args:  cobra.MinimumNArgs(1),
RunE: func(c *cobra.Command, args []string) error {
q := args[0]
client := newSearchClient()

ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
defer cancel()

// Build path; language/sort params may not be supported by all Gitee versions
path := fmt.Sprintf("/search/repositories?q=%s&per_page=%d", q, reposFlags.limit)
if reposFlags.sort != "" {
path += "&sort=" + reposFlags.sort
}
if reposFlags.order != "" {
path += "&order=" + reposFlags.order
}
if reposFlags.language != "" {
path += "&language=" + reposFlags.language
}

var result []interface{}
if err := client.Get(ctx, path, &result); err != nil {
// Gitee may 400 on unsupported params; retry without filters
if result == nil {
path2 := fmt.Sprintf("/search/repositories?q=%s&per_page=%d", q, reposFlags.limit)
_ = client.Get(ctx, path2, &result)
}
}

if cmd.GlobalOutput == "json" {
if result == nil {
result = []interface{}{}
}
b, _ := json.MarshalIndent(result, "", "  ")
fmt.Println(string(b))
return nil
}

// Always show header
w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
fmt.Fprintln(w, "NAME\tLANGUAGE\tSTARS\tDESCRIPTION")
for _, raw := range result {
m, ok := raw.(map[string]interface{})
if !ok {
continue
}
name, _ := m["full_name"].(string)
lang, _ := m["language"].(string)
stars := fmt.Sprintf("%v", m["stargazers_count"])
desc, _ := m["description"].(string)
if len(desc) > 50 {
desc = desc[:47] + "..."
}
fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, lang, stars, desc)
}
_ = w.Flush()
if len(result) == 0 {
fmt.Println("(no results)")
}
return nil
},
}

// ----- issues -----

var searchIssuesCmd = &cobra.Command{
Use:   "issues <query>",
Short: "Search issues",
Args:  cobra.MinimumNArgs(1),
RunE: func(c *cobra.Command, args []string) error {
q := args[0]
client := newSearchClient()
ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
defer cancel()

var result interface{}
path := fmt.Sprintf("/search/issues?q=%s&per_page=20", q)
if err := client.Get(ctx, path, &result); err != nil {
return fmt.Errorf("search issues: %w", err)
}
b, _ := json.MarshalIndent(result, "", "  ")
fmt.Println(string(b))
return nil
},
}

// ----- users -----

var searchUsersCmd = &cobra.Command{
Use:   "users <query>",
Short: "Search users",
Args:  cobra.MinimumNArgs(1),
RunE: func(c *cobra.Command, args []string) error {
q := args[0]
client := newSearchClient()
ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
defer cancel()

var result interface{}
path := fmt.Sprintf("/search/users?q=%s&per_page=20", q)
if err := client.Get(ctx, path, &result); err != nil {
return fmt.Errorf("search users: %w", err)
}
if cmd.GlobalOutput == "json" {
b, _ := json.MarshalIndent(result, "", "  ")
fmt.Println(string(b))
} else {
b, _ := json.MarshalIndent(result, "", "  ")
fmt.Println(string(b))
}
return nil
},
}

func init() {
f := searchReposCmd.Flags()
f.StringVar(&reposFlags.language, "language", "", "Filter by programming language")
f.StringVar(&reposFlags.sort, "sort", "", "Sort by: stars|forks|updated")
f.StringVar(&reposFlags.order, "order", "desc", "Sort order: asc|desc")
f.IntVarP(&reposFlags.limit, "limit", "L", 20, "Maximum number of results")

SearchCmd.AddCommand(searchReposCmd, searchIssuesCmd, searchUsersCmd)
}
