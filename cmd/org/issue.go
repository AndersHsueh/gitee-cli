package org

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
	"github.com/spf13/cobra"
)

var orgIssueFlags struct {
	org    string
	state  string
	labels string
	limit  int
}

var issueCmd = &cobra.Command{
	Use:   "issues <org>",
	Short: "List organization issues",
	Example: `  gitee org issues my-org
  gitee org issues my-org --state closed --labels bug`,
	RunE: runOrgIssues,
}

func init() {
	f := issueCmd.Flags()
	f.StringVar(&orgIssueFlags.org, "org", "", "Organization name")
	f.StringVar(&orgIssueFlags.state, "state", "open", "Filter by state: open|closed|all")
	f.StringVar(&orgIssueFlags.labels, "labels", "", "Filter by labels")
	f.IntVarP(&orgIssueFlags.limit, "limit", "L", 30, "Maximum number of issues")
}

func runOrgIssues(cobraCmd *cobra.Command, args []string) error {
	org := orgIssueFlags.org
	if len(args) > 0 {
		org = args[0]
	}
	if org == "" {
		return fmt.Errorf("organization name is required")
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s?state=%s&per_page=%d", api.OrgIssues(org), orgIssueFlags.state, orgIssueFlags.limit)
	if orgIssueFlags.labels != "" {
		path += "&labels=" + orgIssueFlags.labels
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var issues []types.Issue
	if err := client.Get(ctx, path, &issues); err != nil {
		return fmt.Errorf("listing org issues: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(issues, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(issues) == 0 {
		fmt.Println("No issues found")
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
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		updated := i.UpdatedAt.Format("2006-01-02")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", i.Number, title, i.State, labelStr, updated)
	}
	return w.Flush()
}
