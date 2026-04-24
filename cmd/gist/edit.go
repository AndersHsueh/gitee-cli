package gist

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
)

var editFlags struct {
	description string
	filename   string
	content    string
}

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a gist",
	Example: `  gitee gist edit abc123 --description "New description"`,
	RunE: runEdit,
}

func init() {
	f := editCmd.Flags()
	f.StringVar(&editFlags.description, "description", "", "New description")
	f.StringVar(&editFlags.filename, "filename", "", "Filename to edit")
	f.StringVar(&editFlags.content, "content", "", "New file content")
}

func runEdit(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("gist ID is required")
	}
	gistID := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.GistEditRequest{
		Description: editFlags.description,
	}

	path := api.GistEdit(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Patch(ctx, path, req, nil); err != nil {
		return fmt.Errorf("editing gist: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Gist %s updated\n", green("✓"), gistID)
	return nil
}
