package gist

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/terminal"
	"gitee-cli/pkg/types"
)

var createFlags struct {
	description string
	public     bool
	filename   string
	files      []string
	interactive bool
}

var createCmd = &cobra.Command{
	Use:   "create [files...]",
	Short: "Create a new gist",
	Example: `  gitee gist create file.txt
  gitee gist create file1.txt file2.txt file3.txt
  gitee gist create --description "My snippet" --public file.txt
  cat file.txt | gitee gist create --filename code.go
  gitee gist create --file readme.md="README content" --file main.go="package main"
  gitee gist create --interactive`,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVar(&createFlags.description, "description", "", "Gist description")
	f.BoolVar(&createFlags.public, "public", false, "Make gist public")
	f.StringVar(&createFlags.filename, "filename", "", "Filename for content from stdin")
	f.StringArrayVar(&createFlags.files, "file", []string{}, "File: filename=content or filename=@filepath (can be specified multiple times)")
	f.BoolVarP(&createFlags.interactive, "interactive", "i", false, "Interactive mode")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	files := make(map[string]types.GistFileContent)
	description := createFlags.description
	isPublic := createFlags.public

	// Interactive mode
	if createFlags.interactive {
		fmt.Println("=== Gist Creator (Interactive Mode) ===")

		desc, err := terminal.PromptDefault("Gist description", "My snippet")
		if err != nil {
			return err
		}
		description = desc

		pub, err := terminal.Confirm("Make gist public?")
		if err != nil {
			return err
		}
		isPublic = pub

		fmt.Println("\nAdd files (leave filename empty to finish):")
		for {
			filename, err := terminal.Prompt("Filename (or Enter to finish)")
			if err != nil {
				return err
			}
			if filename == "" {
				break
			}

			content, err := terminal.Prompt(fmt.Sprintf("Content for %s", filename))
			if err != nil {
				return err
			}
			files[filename] = types.GistFileContent{Content: content}

			fmt.Printf("Added: %s\n", filename)
		}

		if len(files) == 0 {
			return fmt.Errorf("no files added")
		}
	} else {
		// Non-interactive mode: Handle --file flags for inline content
		for _, fileFlag := range createFlags.files {
			parts := strings.SplitN(fileFlag, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid --file format: %s (expected filename=content or filename=@filepath)", fileFlag)
			}

			filename := parts[0]
			contentOrPath := parts[1]

			var content string
			if strings.HasPrefix(contentOrPath, "@") {
				filePath := contentOrPath[1:]
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read file %s: %w", filePath, err)
				}
				content = string(data)
			} else {
				content = contentOrPath
			}

			files[filename] = types.GistFileContent{Content: content}
		}

		// Handle positional arguments
		if len(args) > 0 {
			for _, arg := range args {
				content, err := os.ReadFile(arg)
				if err != nil {
					return fmt.Errorf("reading file %s: %w", arg, err)
				}
				files[arg] = types.GistFileContent{Content: string(content)}
			}
		}

		// If no files provided, read from stdin
		if len(files) == 0 {
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

		if len(files) == 0 {
			return fmt.Errorf("no files provided. Use positional arguments, --file flag, --interactive, or stdin")
		}
	}

	req := types.GistCreateRequest{
		Description: description,
		Public:      isPublic,
		Files:       files,
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
	fmt.Printf("  Files: %d\n", len(gist.Files))
	return nil
}
