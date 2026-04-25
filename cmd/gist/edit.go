package gist

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
)

var editFlags struct {
	description string
	files      []string
}

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a gist",
	Example: `  gitee gist edit abc123 --description "New description"
  gitee gist edit abc123 --file readme.md="New README content"
  gitee gist edit abc123 --file file1.txt="Content 1" --file file2.txt="@/path/to/file2.txt"
  gitee gist edit abc123 --file readme.md="Updated" --file newfile.md="New file"`,
	RunE: runEdit,
}

func init() {
	f := editCmd.Flags()
	f.StringVar(&editFlags.description, "description", "", "New description")
	f.StringArrayVar(&editFlags.files, "file", []string{}, "File to edit: filename=content or filename=@filepath (can be specified multiple times)")
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

	// Parse files
	if len(editFlags.files) > 0 {
		req.Files = make(map[string]struct {
			Content  string `json:"content,omitempty"`
			Filename string `json:"filename,omitempty"`
		})

		for _, fileFlag := range editFlags.files {
			// Parse format: filename=content or filename=@filepath
			parts := strings.SplitN(fileFlag, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid --file format: %s (expected filename=content or filename=@filepath)", fileFlag)
			}

			filename := parts[0]
			contentOrPath := parts[1]

			var content string
			if strings.HasPrefix(contentOrPath, "@") {
				// Read content from file
				filePath := contentOrPath[1:]
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read file %s: %w", filePath, err)
				}
				content = string(data)
			} else {
				content = contentOrPath
			}

			req.Files[filename] = struct {
				Content  string `json:"content,omitempty"`
				Filename string `json:"filename,omitempty"`
			}{
				Content: content,
			}
		}
	}

	path := api.GistEdit(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var result types.Gist
	if err := client.Patch(ctx, path, req, &result); err != nil {
		return fmt.Errorf("editing gist: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Gist %s updated\n", green("✓"), gistID)

	// Show updated files if --file was used
	if len(editFlags.files) > 0 {
		fmt.Println("\nUpdated files:")
		for filename := range result.Files {
			fmt.Printf("  - %s\n", filename)
		}
	}

	return nil
}

// GistEditRequestExtended extends the types.GistEditRequest to handle multi-file edits properly
type GistEditRequestExtended struct {
	Description string            `json:"description,omitempty"`
	Files       map[string]struct {
		Content  string `json:"content,omitempty"`
		Filename string `json:"filename,omitempty"`
	} `json:"files,omitempty"`
}

// MarshalJSON implements json.Marshaler for proper multi-file edit format
func (r GistEditRequestExtended) MarshalJSON() ([]byte, error) {
	type alias GistEditRequestExtended
	return json.Marshal((*alias)(&r))
}
