// Package handler contains the business logic shared by multiple command
// packages (e.g. cmd/label/ and cmd/issue/ both call the same label
// handler functions). Putting the logic here means a single behavior
// for "list labels", "create label", etc., and lets the top-level
// `gitee label` command coexist with the legacy `gitee issue label`
// command without code duplication.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
)

// LabelResolveRepo resolves --repo flag / global --repo into (owner, repo).
// Returns a clear error if neither is set or the format is invalid.
// Delegates to utils.ResolveRepoFromFlag so the global --repo fallback
// and parsing logic are shared across all command groups (label, release,
// webhook).
func LabelResolveRepo(repo string) (string, string, error) {
	return utils.ResolveRepoFromFlag(repo, cmd.GlobalRepo)
}

// IsValidHexColor checks if s is exactly 6 lowercase hex digits.
func IsValidHexColor(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// PrintJSON marshals v and writes it to stdout.
func PrintJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// LabelList runs the label-list operation.
func LabelList(repo, filter string, limit int) error {
	owner, repoName, err := LabelResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.LabelList(owner, repoName)
	q := url.Values{}
	q.Set("per_page", fmt.Sprintf("%d", limit))
	if filter != "" {
		q.Set("color", filter)
	}
	fullPath := path + "?" + q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var labels []types.Label
	if err := client.Get(ctx, fullPath, &labels); err != nil {
		return fmt.Errorf("listing labels: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return PrintJSON(labels)
	}

	if len(labels) == 0 {
		fmt.Println("No labels found in this repository")
		return nil
	}

	w := output.NewTableWriter()
	fmt.Fprintln(w, "NAME\tCOLOR\tURL")
	for _, l := range labels {
		colorHex := l.Color
		if !strings.HasPrefix(colorHex, "#") {
			colorHex = "#" + colorHex
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", l.Name, colorHex, l.URL)
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d label(s)\n", len(labels))
	return nil
}

// LabelView runs the label-view operation.
func LabelView(repo, name string) error {
	if name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}
	owner, repoName, err := LabelResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	encodedName := url.QueryEscape(name)
	path := api.LabelGet(owner, repoName, encodedName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var label types.Label
	if err := client.Get(ctx, path, &label); err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("label %q not found in %s", name, repoName)
		}
		return fmt.Errorf("getting label: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return PrintJSON(label)
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
func LabelCreate(repo, name, labelColor string) error {
	if name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}
	owner, repoName, err := LabelResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var label types.Label
	if err := client.Post(ctx, path, req, &label); err != nil {
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
func LabelEdit(repo, currentName, newName, labelColor string) error {
	if currentName == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}
	if newName == "" && labelColor == "" {
		return fmt.Errorf("at least one of --new-name or --color is required")
	}
	owner, repoName, err := LabelResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var label types.Label
	if err := client.Patch(ctx, path, req, &label); err != nil {
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
func LabelDelete(repo, name, confirm string) error {
	if name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}
	owner, repoName, err := LabelResolveRepo(repo)
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

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	encodedName := url.QueryEscape(name)
	path := api.LabelDelete(owner, repoName, encodedName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("deleting label: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n%s Label %q deleted successfully\n", green("✓"), name)
	return nil
}

// LabelClone copies the entire label set from srcRepo to dstRepo.
// Useful for bootstrapping a new repo with a consistent taxonomy.
func LabelClone(srcRepo, dstRepo string) error {
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

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	// Per-call context: each label POST gets its own timeout so a long clone
	// of 50+ labels doesn't blow through one shared deadline. The previous
	// version used one shared context, which caused the second half of large
	// clones to fail with "context deadline exceeded" before completing.
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
		var srcLabels []types.Label
		err := client.Get(ctx, api.LabelList(srcOwner, srcName), &srcLabels)
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
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
			req := types.LabelCreateRequest{Name: l.Name, Color: l.Color}
			var createdLabel types.Label
			err := client.Post(ctx, api.LabelCreate(dstOwner, dstName), req, &createdLabel)
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
