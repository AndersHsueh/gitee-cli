package issue

import (
"context"
"encoding/json"
"fmt"
"os"
"strings"
"text/tabwriter"
"time"

"github.com/spf13/cobra"
"gitee-cli/cmd"
"gitee-cli/pkg/types"
)

var listFlags struct {
state     string
labels    string
owner     string
repo      string
assignee  string
milestone string
limit     int
}

var listCmd = &cobra.Command{
Use:   "list",
Short: "List issues in a repository",
Example: `  gitee issue list --repo owner/repo
  gitee issue list --repo owner/repo --state closed --labels bug`,
RunE: runList,
}

func init() {
f := listCmd.Flags()
f.StringVar(&listFlags.state, "state", "open", "Filter by state: open|closed|all")
f.StringVar(&listFlags.labels, "labels", "", "Filter by labels (comma-separated)")
f.StringVar(&listFlags.owner, "owner", "", "Repository owner")
f.StringVarP(&listFlags.repo, "repo", "R", "", "Repository name or owner/repo")
f.StringVar(&listFlags.assignee, "assignee", "", "Filter by assignee username")
f.StringVar(&listFlags.milestone, "milestone", "", "Filter by milestone")
f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of issues to list")
}

func runList(cobraCmd *cobra.Command, args []string) error {
owner, repoName, err := resolveRepo(listFlags.owner, listFlags.repo)
if err != nil {
return err
}

client, err := cmd.NewAPIClient()
if err != nil {
return err
}

path := fmt.Sprintf("/repos/%s/%s/issues?state=%s&per_page=%d",
owner, repoName, listFlags.state, listFlags.limit)
if listFlags.labels != "" {
path += "&labels=" + listFlags.labels
}
if listFlags.assignee != "" {
path += "&assignee=" + listFlags.assignee
}
if listFlags.milestone != "" {
path += "&milestone=" + listFlags.milestone
}

ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
defer cancel()

var issues []types.Issue
if err := client.Get(ctx, path, &issues); err != nil {
return fmt.Errorf("listing issues: %w", err)
}

if cmd.GlobalOutput == "json" {
b, _ := json.MarshalIndent(issues, "", "  ")
fmt.Println(string(b))
return nil
}

if len(issues) == 0 {
fmt.Printf("No %s issues found in %s/%s\n", listFlags.state, owner, repoName)
return nil
}

w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
fmt.Fprintln(w, "#\tTITLE\tSTATE\tLABELS\tUPDATED")
for _, i := range issues {
lbls := make([]string, 0, len(i.Labels))
for _, l := range i.Labels {
lbls = append(lbls, l.Name)
}
labelStr := strings.Join(lbls, ",")
if labelStr == "" {
labelStr = "-"
}
title := i.Title
if len(title) > 60 {
title = title[:57] + "..."
}
updated := i.UpdatedAt.Format("2006-01-02")
fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", i.Number, title, i.State, labelStr, updated)
}
_ = w.Flush()
return nil
}
