package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
)

// Label command flags
var labelListFlags struct {
	owner  string
	repo   string
	limit  int
	filter string
}

var labelViewFlags struct {
	owner string
	repo  string
	name  string
}

var labelCreateFlags struct {
	owner      string
	repo       string
	name       string
	labelColor string
}

var labelUpdateFlags struct {
	owner      string
	repo       string
	name       string
	newName    string
	labelColor string
}

var labelDeleteFlags struct {
	owner string
	repo  string
	name  string
}

// Command definitions
var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Manage issue labels",
	Long: `Manage issue labels for a repository.

Labels are used to categorize and organize issues. You can create, list,
view details, update, and delete labels.

Examples:
  gitee issue label list --repo owner/repo
  gitee issue label create --repo owner/repo --name bug --color ff0000
  gitee issue label view --repo owner/repo --name bug
  gitee issue label update --repo owner/repo --name bug --color 00ff00
  gitee issue label delete --repo owner/repo --name bug`,
}

var labelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List labels in a repository",
	Example: `  gitee issue label list --repo owner/repo
  gitee issue label list --repo owner/repo --limit 50`,
	RunE:  runLabelList,
}

var labelViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View label details",
	Example: `  gitee issue label view --repo owner/repo --name bug
  gitee issue label view -R owner/repo -n enhancement`,
	RunE:  runLabelView,
}

var labelCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new label",
	Example: `  gitee issue label create --repo owner/repo --name bug --color ff0000
  gitee issue label create --repo owner/repo --name enhancement --color 00ff00`,
	RunE:  runLabelCreate,
}

var labelUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing label",
	Example: `  gitee issue label update --repo owner/repo --name bug --color 00ff00
  gitee issue label update --repo owner/repo --name bug --new-name fixed-bug`,
	RunE:  runLabelUpdate,
}

var labelDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a label",
	Example: `  gitee issue label delete --repo owner/repo --name bug
  gitee issue label delete -R owner/repo -n wontfix`,
	RunE:  runLabelDelete,
}

func init() {
	// IssueCmd.AddCommand(labelCmd) is already called in issue.go

	// List command flags
	labelListCmd.Flags().StringVar(&labelListFlags.owner, "owner", "", "Repository owner")
	labelListCmd.Flags().StringVarP(&labelListFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	labelListCmd.Flags().IntVarP(&labelListFlags.limit, "limit", "L", 100, "Maximum number of labels to fetch")
	labelListCmd.Flags().StringVar(&labelListFlags.filter, "color", "", "Filter labels by color")

	// View command flags
	labelViewCmd.Flags().StringVar(&labelViewFlags.owner, "owner", "", "Repository owner")
	labelViewCmd.Flags().StringVarP(&labelViewFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	labelViewCmd.Flags().StringVarP(&labelViewFlags.name, "name", "n", "", "Label name (required)")

	// Create command flags
	labelCreateCmd.Flags().StringVar(&labelCreateFlags.owner, "owner", "", "Repository owner")
	labelCreateCmd.Flags().StringVarP(&labelCreateFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	labelCreateCmd.Flags().StringVarP(&labelCreateFlags.name, "name", "n", "", "Label name (required)")
	labelCreateCmd.Flags().StringVarP(&labelCreateFlags.labelColor, "color", "c", "gray", "Label color (6 hex digits, e.g. ff0000)")

	// Update command flags
	labelUpdateCmd.Flags().StringVar(&labelUpdateFlags.owner, "owner", "", "Repository owner")
	labelUpdateCmd.Flags().StringVarP(&labelUpdateFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	labelUpdateCmd.Flags().StringVarP(&labelUpdateFlags.name, "name", "n", "", "Current label name (required)")
	labelUpdateCmd.Flags().StringVar(&labelUpdateFlags.newName, "new-name", "", "New label name")
	labelUpdateCmd.Flags().StringVarP(&labelUpdateFlags.labelColor, "color", "c", "", "New label color")

	// Delete command flags
	labelDeleteCmd.Flags().StringVar(&labelDeleteFlags.owner, "owner", "", "Repository owner")
	labelDeleteCmd.Flags().StringVarP(&labelDeleteFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	labelDeleteCmd.Flags().StringVarP(&labelDeleteFlags.name, "name", "n", "", "Label name to delete (required)")

	labelCmd.AddCommand(labelListCmd, labelViewCmd, labelCreateCmd, labelUpdateCmd, labelDeleteCmd)
}

func runLabelList(c *cobra.Command, args []string) error {
	owner, repoName, err := resolveLabelRepo(labelListFlags.owner, labelListFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.LabelList(owner, repoName)
	query := url.Values{}
	query.Set("per_page", fmt.Sprintf("%d", labelListFlags.limit))
	if labelListFlags.filter != "" {
		query.Set("color", labelListFlags.filter)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var labels []types.Label
	fullPath := path + "?" + query.Encode()
	if err := client.Get(ctx, fullPath, &labels); err != nil {
		return fmt.Errorf("listing labels: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return printJSON(labels)
	}

	if len(labels) == 0 {
		fmt.Println("No labels found in this repository")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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

func runLabelView(c *cobra.Command, args []string) error {
	if labelViewFlags.name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}

	owner, repoName, err := resolveLabelRepo(labelViewFlags.owner, labelViewFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	// URL encode the label name
	encodedName := url.QueryEscape(labelViewFlags.name)
	path := api.LabelGet(owner, repoName, encodedName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var label types.Label
	if err := client.Get(ctx, path, &label); err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("label %q not found in %s/%s", labelViewFlags.name, owner, repoName)
		}
		return fmt.Errorf("getting label: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return printJSON(label)
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

func runLabelCreate(c *cobra.Command, args []string) error {
	if labelCreateFlags.name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}

	owner, repoName, err := resolveLabelRepo(labelCreateFlags.owner, labelCreateFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	// Validate and normalize color format
	labelColor := strings.ToLower(strings.TrimPrefix(labelCreateFlags.labelColor, "#"))
	if !isValidHexColor(labelColor) {
		return fmt.Errorf("invalid color format: %s (expected 6 hex digits, e.g. ff0000)", labelCreateFlags.labelColor)
	}

	req := types.LabelCreateRequest{
		Name:  labelCreateFlags.name,
		Color: labelColor,
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

func runLabelUpdate(c *cobra.Command, args []string) error {
	if labelUpdateFlags.name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}

	if labelUpdateFlags.newName == "" && labelUpdateFlags.labelColor == "" {
		return fmt.Errorf("at least one of --new-name or --color is required")
	}

	owner, repoName, err := resolveLabelRepo(labelUpdateFlags.owner, labelUpdateFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	// Validate and normalize color if provided
	labelColor := labelUpdateFlags.labelColor
	if labelColor != "" {
		labelColor = strings.ToLower(strings.TrimPrefix(labelColor, "#"))
		if !isValidHexColor(labelColor) {
			return fmt.Errorf("invalid color format: %s (expected 6 hex digits, e.g. ff0000)", labelUpdateFlags.labelColor)
		}
	}

	req := types.LabelUpdateRequest{
		Name:  labelUpdateFlags.newName,
		Color: labelColor,
	}

	// Use current name for the API endpoint
	encodedName := url.QueryEscape(labelUpdateFlags.name)
	path := api.LabelGet(owner, repoName, encodedName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var label types.Label
	// Gitee API uses PATCH for update
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

func runLabelDelete(c *cobra.Command, args []string) error {
	if labelDeleteFlags.name == "" {
		return fmt.Errorf("label name is required (use --name or -n)")
	}

	owner, repoName, err := resolveLabelRepo(labelDeleteFlags.owner, labelDeleteFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	// Confirm deletion
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s Delete label %q from %s/%s? [y/N] ", yellow("?"), labelDeleteFlags.name, owner, repoName)

	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
		fmt.Println("Cancelled")
		return nil
	}

	// URL encode the label name
	encodedName := url.QueryEscape(labelDeleteFlags.name)
	path := api.LabelDelete(owner, repoName, encodedName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("deleting label: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n%s Label %q deleted successfully\n", green("✓"), labelDeleteFlags.name)

	return nil
}

func resolveLabelRepo(owner, repo string) (string, string, error) {
	combined := repo
	if combined == "" {
		combined = cmd.GlobalRepo
	}
	if combined == "" {
		return "", "", fmt.Errorf("--repo is required (e.g. --repo owner/repo or --owner owner --repo name)")
	}

	parts := strings.SplitN(combined, "/", 2)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}
	if owner != "" {
		return owner, combined, nil
	}
	return "", "", fmt.Errorf("invalid --repo format %q, expected owner/repo", combined)
}

func isValidHexColor(s string) bool {
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

func printJSON(v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
