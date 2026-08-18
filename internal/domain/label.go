package domain

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
)

// IsValidHexColor checks if s is exactly 6 lowercase hex digits.
func IsValidHexColor(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// LabelList runs the label-list operation.
func (c *Client) LabelList(repo, filter string, limit int) error {
	return cmdkit.RunList[types.Label](c.opts(), cmdkit.ListSpec[types.Label]{
		ResolveOwnerRepo: func() (string, string, error) {
			return c.ResolveRepo(repo)
		},
		Path: func(owner, repoName string) string {
			path := api.LabelList(owner, repoName)
			q := url.Values{}
			q.Set("per_page", fmt.Sprintf("%d", limit))
			if filter != "" {
				q.Set("color", filter)
			}
			return path + "?" + q.Encode()
		},
		// label list caps per_page with --limit and does a single GET — no
		// --paginate support.
		Fetch: func(ctx context.Context, client *api.Client, path string) ([]types.Label, error) {
			var labels []types.Label
			if err := client.Get(ctx, path, &labels); err != nil {
				return nil, fmt.Errorf("listing labels: %w", err)
			}
			return labels, nil
		},
		Columns: []string{"NAME", "COLOR", "URL"},
		Row: func(l types.Label) []string {
			colorHex := l.Color
			if !strings.HasPrefix(colorHex, "#") {
				colorHex = "#" + colorHex
			}
			return []string{l.Name, colorHex, l.URL}
		},
		EmptyText: func(_, _ string) string {
			return "No labels found in this repository"
		},
		Footer: func(count int) string {
			return fmt.Sprintf("\nTotal: %d label(s)\n", count)
		},
		JSON: func(items []types.Label) error {
			return output.PrintJSON(items)
		},
	})
}

// LabelView runs the label-view operation.
func (c *Client) LabelView(repo, name string) error {
	if name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}
	owner, repoName, err := c.ResolveRepo(repo)
	if err != nil {
		return err
	}

	encodedName := url.QueryEscape(name)
	path := api.LabelGet(owner, repoName, encodedName)

	ctx, cancel := c.ctx()
	defer cancel()

	var label types.Label
	if err := c.API.Get(ctx, path, &label); err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("label %q not found in %s", name, repoName)
		}
		return fmt.Errorf("getting label: %w", err)
	}

	if c.IsJSON() {
		return output.PrintJSON(label)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Label Details\n", bold("─"))
	fmt.Printf("  Name:  %s\n", green(label.Name))

	colorHex := label.Color
	if !strings.HasPrefix(colorHex, "#") {
		colorHex = "#" + colorHex
	}
	cAttr := color.New(color.BgHiRed).Add(color.FgWhite)
	fmt.Printf("  Color: %s (%s)\n", colorHex, cAttr.Sprint("  "))
	if label.Description != "" {
		fmt.Printf("  Desc:  %s\n", label.Description)
	}
	fmt.Printf("  URL:   %s\n", label.URL)
	fmt.Println()
	return nil
}

// LabelCreate runs the label-create operation.
func (c *Client) LabelCreate(repo, name, labelColor string) error {
	if name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}
	owner, repoName, err := c.ResolveRepo(repo)
	if err != nil {
		return err
	}

	normalized := strings.ToLower(strings.TrimPrefix(labelColor, "#"))
	if !IsValidHexColor(normalized) {
		return fmt.Errorf("invalid color format: %s (expected 6 hex digits, e.g. ff0000)", labelColor)
	}

	req := types.LabelCreateRequest{
		Name:  name,
		Color: normalized,
	}

	path := api.LabelCreate(owner, repoName)
	ctx, cancel := c.ctx()
	defer cancel()

	var label types.Label
	if err := c.API.Post(ctx, path, req, &label); err != nil {
		return fmt.Errorf("creating label: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Label created successfully\n", green("✓"))
	fmt.Printf("  Name:  %s\n", bold(label.Name))
	colorHex := label.Color
	if !strings.HasPrefix(colorHex, "#") {
		colorHex = "#" + colorHex
	}
	fmt.Printf("  Color: %s\n", colorHex)
	if label.URL != "" {
		fmt.Printf("  URL:   %s\n", label.URL)
	}
	fmt.Println()
	return nil
}

// LabelEdit runs the label-update operation (Gitee API uses PATCH).
func (c *Client) LabelEdit(repo, currentName, newName, labelColor string) error {
	if currentName == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}
	if newName == "" && labelColor == "" {
		return fmt.Errorf("at least one of --new-name or --color is required")
	}
	owner, repoName, err := c.ResolveRepo(repo)
	if err != nil {
		return err
	}

	normalized := labelColor
	if normalized != "" {
		normalized = strings.ToLower(strings.TrimPrefix(normalized, "#"))
		if !IsValidHexColor(normalized) {
			return fmt.Errorf("invalid color format: %s (expected 6 hex digits, e.g. ff0000)", labelColor)
		}
	}

	req := types.LabelUpdateRequest{
		Name:  newName,
		Color: normalized,
	}

	encodedName := url.QueryEscape(currentName)
	path := api.LabelGet(owner, repoName, encodedName)

	ctx, cancel := c.ctx()
	defer cancel()

	var label types.Label
	if err := c.API.Patch(ctx, path, req, &label); err != nil {
		return fmt.Errorf("updating label: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Label updated successfully\n", green("✓"))
	fmt.Printf("  Name:  %s\n", bold(label.Name))
	colorHex := label.Color
	if !strings.HasPrefix(colorHex, "#") {
		colorHex = "#" + colorHex
	}
	fmt.Printf("  Color: %s\n", colorHex)
	if label.URL != "" {
		fmt.Printf("  URL:   %s\n", label.URL)
	}
	fmt.Println()
	return nil
}

// LabelDelete runs the label-delete operation. If confirm is non-empty and
// the user typed "y"/"yes", the delete proceeds. Otherwise it is cancelled.
// In non-interactive mode (e.g. piped stdin), pass confirm="y" to skip.
func (c *Client) LabelDelete(repo, name, confirm string) error {
	if name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}
	owner, repoName, err := c.ResolveRepo(repo)
	if err != nil {
		return err
	}

	if confirm != "y" && confirm != "Y" && confirm != "yes" {
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Delete label %q from %s/%s?", color.YellowString("?"), name, owner, repoName),
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

	encodedName := url.QueryEscape(name)
	path := api.LabelDelete(owner, repoName, encodedName)

	ctx, cancel := c.ctx()
	defer cancel()

	if err := c.API.Delete(ctx, path); err != nil {
		return fmt.Errorf("deleting label: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n%s Label %q deleted successfully\n", green("✓"), name)
	return nil
}

// LabelClone copies the entire label set from srcRepo to dstRepo.
// Useful for bootstrapping a new repo with a consistent taxonomy.
func (c *Client) LabelClone(srcRepo, dstRepo string) error {
	if srcRepo == "" || dstRepo == "" {
		return fmt.Errorf("both --source and --target are required (owner/repo)")
	}
	srcOwner, srcName, err := utils.ResolveRepo("", srcRepo, nil)
	if err != nil {
		return fmt.Errorf("invalid --source: %w", err)
	}
	dstOwner, dstName, err := utils.ResolveRepo("", dstRepo, nil)
	if err != nil {
		return fmt.Errorf("invalid --target: %w", err)
	}

	// Per-call context: each label POST gets its own timeout so a long clone
	// of 50+ labels doesn't blow through one shared deadline.
	{
		ctx, cancel := c.ctx()
		var srcLabels []types.Label
		err := c.API.Get(ctx, api.LabelList(srcOwner, srcName), &srcLabels)
		cancel()
		if err != nil {
			return fmt.Errorf("listing source labels: %w", err)
		}
		if len(srcLabels) == 0 {
			fmt.Printf("No labels to clone from %s/%s\n", srcOwner, srcName)
			return nil
		}
		green := color.New(color.FgGreen).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		created, failed := 0, 0
		for _, l := range srcLabels {
			// Fresh context per label so the deadline doesn't expire mid-loop.
			ctx, cancel := c.ctx()
			req := types.LabelCreateRequest{Name: l.Name, Color: l.Color}
			var createdLabel types.Label
			err := c.API.Post(ctx, api.LabelCreate(dstOwner, dstName), req, &createdLabel)
			cancel()
			if err != nil {
				// Likely 409 (label already exists on target). Don't abort; tally.
				failed++
				fmt.Fprintf(os.Stderr, "%s %s\n", red("✗"), l.Name)
				continue
			}
			created++
			fmt.Printf("%s %s\n", green("✓"), l.Name)
		}
		fmt.Printf("\nCloned %d label(s) from %s/%s to %s/%s (%d failed, probably duplicates)\n",
			created, srcOwner, srcName, dstOwner, dstName, failed)
	}
	return nil
}
