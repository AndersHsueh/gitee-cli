// Package handler — milestone subpackage logic.
//
// Milestone handlers follow the same shape as the label / webhook handlers:
// per-repo, list / view / create / edit / delete. Milestones are addressed
// by an integer `number` (not a string name like labels), so URL escaping
// is unnecessary — the only path segment is the number.
package handler

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
)

// MilestoneResolveRepo resolves --repo flag / global --repo into (owner, repo).
// Delegates to utils.ResolveRepoFromFlag so the global --repo fallback
// and parsing logic are shared across all command groups.
func MilestoneResolveRepo(repo string) (string, string, error) {
	return utils.ResolveRepoFromFlag(repo, cmd.GlobalRepo)
}

// MilestoneList runs the milestone-list operation.
func MilestoneList(repo, stateFilter string, limit int) error {
	owner, repoName, err := MilestoneResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.MilestonesList(owner, repoName)
	q := url.Values{}
	q.Set("per_page", fmt.Sprintf("%d", limit))
	if stateFilter != "" {
		q.Set("state", stateFilter)
	}
	fullPath := path + "?" + q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var milestones []types.Milestone
	if err := client.Get(ctx, fullPath, &milestones); err != nil {
		return fmt.Errorf("listing milestones: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return PrintJSON(milestones)
	}

	if len(milestones) == 0 {
		fmt.Printf("No milestones found in %s/%s\n", owner, repoName)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NUMBER\tTITLE\tSTATE\tOPEN\tCLOSED\tDUE")
	for _, m := range milestones {
		due := ""
		if m.DueOn != nil {
			due = m.DueOn.Format("2006-01-02")
		}
		title := utils.Truncate(m.Title, 40)
		fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%s\n",
			m.Number, title, m.State, m.OpenIssues, m.ClosedIssues, due)
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d milestone(s)\n", len(milestones))
	return nil
}

// MilestoneView runs the milestone-view operation.
func MilestoneView(repo string, number int) error {
	if number <= 0 {
		return fmt.Errorf("milestone number is required (positional or --number)")
	}
	owner, repoName, err := MilestoneResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var m types.Milestone
	if err := client.Get(ctx, api.MilestoneGet(owner, repoName, number), &m); err != nil {
		if api.IsNotFound(err) {
			return fmt.Errorf("milestone #%d not found in %s/%s", number, owner, repoName)
		}
		return fmt.Errorf("getting milestone: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return PrintJSON(m)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Milestone Details (number=%s)\n", bold("─"), bold(fmt.Sprintf("%d", m.Number)))
	fmt.Printf("  Title:  %s\n", green(m.Title))
	fmt.Printf("  State:  %s\n", yellow(m.State))
	fmt.Printf("  Open:   %d\n", m.OpenIssues)
	fmt.Printf("  Closed: %d\n", m.ClosedIssues)
	if m.Description != "" {
		fmt.Printf("  Desc:   %s\n", m.Description)
	}
	if m.DueOn != nil {
		fmt.Printf("  Due:    %s\n", m.DueOn.Format("2006-01-02"))
	}
	if m.HTMLURL != "" {
		fmt.Printf("  URL:    %s\n", m.HTMLURL)
	}
	fmt.Println()
	return nil
}

// MilestoneCreate runs the milestone-create operation.
func MilestoneCreate(repo string, req types.MilestoneCreateRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("milestone title is required (use --title or -t)")
	}
	if req.State != "" && req.State != "open" && req.State != "closed" {
		return fmt.Errorf("invalid --state %q: must be 'open' or 'closed'", req.State)
	}
	owner, repoName, err := MilestoneResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var m types.Milestone
	if err := client.Post(ctx, api.MilestoneCreate(owner, repoName), req, &m); err != nil {
		return fmt.Errorf("creating milestone: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Milestone created successfully\n", green("✓"))
	fmt.Printf("  Number: %s\n", bold(fmt.Sprintf("%d", m.Number)))
	fmt.Printf("  Title:  %s\n", green(m.Title))
	if m.HTMLURL != "" {
		fmt.Printf("  URL:    %s\n", m.HTMLURL)
	}
	fmt.Println()
	return nil
}

// MilestoneEdit runs the milestone-update operation (Gitee API uses PATCH).
// At least one updatable field must be supplied; otherwise there is nothing
// for the server to change.
func MilestoneEdit(repo string, number int, req types.MilestoneUpdateRequest) error {
	if number <= 0 {
		return fmt.Errorf("milestone number is required (positional or --number)")
	}
	if req.Title == "" && req.Description == "" && req.DueOn == "" && req.State == "" {
		return fmt.Errorf("at least one of --title/--description/--due-on/--state is required")
	}
	if req.State != "" && req.State != "open" && req.State != "closed" {
		return fmt.Errorf("invalid --state %q: must be 'open' or 'closed'", req.State)
	}
	owner, repoName, err := MilestoneResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var m types.Milestone
	if err := client.Patch(ctx, api.MilestoneUpdate(owner, repoName, number), req, &m); err != nil {
		return fmt.Errorf("updating milestone: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Milestone updated (number=%s)\n", green("✓"), bold(fmt.Sprintf("%d", m.Number)))
	fmt.Printf("  Title:  %s\n", green(m.Title))
	fmt.Printf("  State:  %s\n", m.State)
	fmt.Println()
	return nil
}

// MilestoneDelete runs the milestone-delete operation. confirm matches the
// label-delete / webhook-delete convention: pass "y" (or set --force) to
// skip the y/N prompt. Deletion is permanent; issues that referenced the
// milestone are simply detached (not deleted).
func MilestoneDelete(repo string, number int, confirm string) error {
	if number <= 0 {
		return fmt.Errorf("milestone number is required (positional or --number)")
	}
	owner, repoName, err := MilestoneResolveRepo(repo)
	if err != nil {
		return err
	}

	if confirm != "y" && confirm != "Y" && confirm != "yes" {
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Delete milestone #%d from %s/%s?", color.YellowString("?"), number, owner, repoName),
			false,
		)
		if err != nil {
			return err
		}
		if !yes {
			fmt.Println("Cancelled")
			return nil
		}
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, api.MilestoneDelete(owner, repoName, number)); err != nil {
		return fmt.Errorf("deleting milestone: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n%s Milestone #%d deleted\n", green("✓"), number)
	return nil
}