package gist

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
)

var createFlags struct {
	description string
	public     bool
	filename   string
}

var createCmd = &cobra.Command{
	Use:   "create [file]",
	Short: "Create a new gist",
	Example: `  gitee gist create file.txt
  gitee gist create --description "My snippet" --public file.txt
  cat file.txt | gitee gist create --filename code.go`,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVar(&createFlags.description, "description", "", "Gist description")
	f.BoolVar(&createFlags.public, "public", false, "Make gist public")
	f.StringVar(&createFlags.filename, "filename", "", "Filename for content from stdin")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	files := make(map[string]types.GistFileContent)

	if len(args) > 0 {
		// Read from file
		content, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		filename := args[0]
		files[filename] = types.GistFileContent{Content: string(content)}
	} else {
		// Read from stdin
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		filename := createFlags.filename
		if filename == "" {
			filename = "untitled.txt"
		}
		files[filename] = types.GistFileContent{Content: string(content)}
	}

	req := types.GistCreateRequest{
		Description: createFlags.description,
		Public:     createFlags.public,
		Files:      files,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var gist types.Gist
	if err := client.Post(ctx, api.GistsCreate(), req, &gist); err != nil {
		return fmt.Errorf("creating gist: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Gist created: %s\n", green("✓"), gist.ID)
	fmt.Printf("  URL: %s\n", color.CyanString(gist.HTMLURL))
	return nil
}
