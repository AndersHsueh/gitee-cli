package gist

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var viewFlags struct {
	stdin bool
}

var viewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a gist",
	Example: `  gitee gist view abc123def
  gitee gist view abc123def --raw`,
	RunE: runView,
}

func init() {
	viewCmd.Flags().BoolVar(&viewFlags.stdin, "raw", false, "Output raw file content")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("gist ID is required")
	}
	gistID := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.GistGet(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var gist types.Gist
	if err := client.Get(ctx, path, &gist); err != nil {
		return fmt.Errorf("viewing gist: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(gist, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("ID: %s\n", gist.ID)
	fmt.Printf("Description: %s\n", gist.Description)
	fmt.Printf("Public: %v\n", gist.Public)
	fmt.Printf("Files: %d\n", len(gist.Files))
	fmt.Printf("URL: %s\n", gist.HTMLURL)

	for name, file := range gist.Files {
		fmt.Printf("\n--- %s (%s, %d bytes) ---\n", name, file.Language, file.Size)
		if viewFlags.stdin {
			// Get raw content from raw_url
			if file.RawURL != "" {
				rawResp, _, err := client.GetRaw(ctx, file.RawURL)
				if err == nil {
					fmt.Print(string(rawResp))
				}
			}
		} else {
			// Show truncated content
			content := file.Content
			if content == "" && file.RawURL != "" {
				rawResp, _, err := client.GetRaw(ctx, file.RawURL)
				if err == nil {
					content = string(rawResp)
				}
			}
			content = utils.Truncate(content, 100)
			fmt.Println(content)
		}
	}
	return nil
}
