package domain

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
)

// MilestoneList runs the milestone-list operation.
func (c *Client) MilestoneList(repo, stateFilter string, limit int) error {
	return cmdkit.RunList[types.Milestone](c.opts(), cmdkit.ListSpec[types.Milestone]{
		ResolveOwnerRepo: func() (string, string, error) {
			return c.ResolveRepo(repo)
		},
		Path: func(owner, repoName string) string {
			path := api.MilestonesList(owner, repoName)
			q := url.Values{}
			q.Set("per_page", fmt.Sprintf("%d", limit))
			if stateFilter != "" {
				q.Set("state", stateFilter)
			}
			return path + "?" + q.Encode()
		},
		// milestone list caps per_page with --limit and does a single GET —
		// no --paginate support.
		Fetch: func(ctx context.Context, client *api.Client, path string) ([]types.Milestone, error) {
			var milestones []types.Milestone
			if err := client.Get(ctx, path, &milestones); err != nil {
				return nil, fmt.Errorf("listing milestones: %w", err)
			}
			return milestones, nil
		},
		Columns: []string{"NUMBER", "TITLE", "STATE", "OPEN", "CLOSED", "DUE"},
		Row: func(m types.Milestone) []string {
			due := ""
			if m.DueOn != nil {
				due = m.DueOn.Format("2006-01-02")
			}
			return []string{
				strconv.Itoa(m.Number),
				utils.Truncate(m.Title, 40),
				m.State,
				strconv.Itoa(m.OpenIssues),
				strconv.Itoa(m.ClosedIssues),
				due,
			}
		},
		EmptyText: func(o, r string) string {
			return fmt.Sprintf("No milestones found in %s/%s", o, r)
		},
		Footer: func(count int) string {
			return fmt.Sprintf("\nTotal: %d milestone(s)\n", count)
		},
		JSON: func(items []types.Milestone) error {
			return output.PrintJSON(items)
		},
	})
}

// MilestoneView runs the milestone-view operation.
func (c *Client) MilestoneView(repo string, number int) error {
	if number <= 0 {
		return fmt.Errorf("milestone number is required (positional or --number)")
	}
	owner, repoName, err := c.ResolveRepo(repo)
	if err != nil {
		return err
	}

	ctx, cancel := c.ctx()
	defer cancel()

	var m types.Milestone
	if err := c.API.Get(ctx, api.MilestoneGet(owner, repoName, number), &m); err != nil {
		if api.IsNotFound(err) {
			return fmt.Errorf("milestone #%d not found in %s/%s", number, owner, repoName)
		}
		return fmt.Errorf("getting milestone: %w", err)
	}

	if c.IsJSON() {
		return output.PrintJSON(m)
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
func (c *Client) MilestoneCreate(repo string, req types.MilestoneCreateRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("milestone title is required (use --title or -t)")
	}
	if req.State != "" && req.State != "open" && req.State != "closed" {
		return fmt.Errorf("invalid --state %q: must be 'open' or 'closed'", req.State)
	}
	owner, repoName, err := c.ResolveRepo(repo)
	if err != nil {
		return err
	}

	ctx, cancel := c.ctx()
	defer cancel()

	var m types.Milestone
	if err := c.API.Post(ctx, api.MilestoneCreate(owner, repoName), req, &m); err != nil {
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
func (c *Client) MilestoneEdit(repo string, number int, req types.MilestoneUpdateRequest) error {
	if number <= 0 {
		return fmt.Errorf("milestone number is required (positional or --number)")
	}
	if req.Title == "" && req.Description == "" && req.DueOn == "" && req.State == "" {
		return fmt.Errorf("at least one of --title/--description/--due-on/--state is required")
	}
	if req.State != "" && req.State != "open" && req.State != "closed" {
		return fmt.Errorf("invalid --state %q: must be 'open' or 'closed'", req.State)
	}
	owner, repoName, err := c.ResolveRepo(repo)
	if err != nil {
		return err
	}

	ctx, cancel := c.ctx()
	defer cancel()

	var m types.Milestone
	if err := c.API.Patch(ctx, api.MilestoneUpdate(owner, repoName, number), req, &m); err != nil {
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
func (c *Client) MilestoneDelete(repo string, number int, confirm string) error {
	if number <= 0 {
		return fmt.Errorf("milestone number is required (positional or --number)")
	}
	owner, repoName, err := c.ResolveRepo(repo)
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

	ctx, cancel := c.ctx()
	defer cancel()

	if err := c.API.Delete(ctx, api.MilestoneDelete(owner, repoName, number)); err != nil {
		return fmt.Errorf("deleting milestone: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n%s Milestone #%d deleted\n", green("✓"), number)
	return nil
}
