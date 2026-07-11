package gist

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/terminal"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var createFlags struct {
	description string
	public      bool
	filename    string
	files       []string
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
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gistCreateAgentHelp)
		}
		return nil
	},
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
			filename, content, err := utils.ParseGistFileFlag(fileFlag)
			if err != nil {
				return err
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

// gistCreateAgentHelp is the agent-targeted help for `gitee gist create`.
// Maintainer note: keep this in sync with the actual command behavior.
const gistCreateAgentHelp = `# gitee gist create — Agent Help

## Purpose
Create a new gist on Gitee from local files, inline content, or stdin.
Use this when an agent needs to publish a snippet (e.g. a repro script,
a config example, a one-off tool) on behalf of the user.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- At least one file content source must be available (positional arg,
  --file flag, --interactive, or stdin)
- For --public: gists are private by default; opt in explicitly

## Required arguments
  none — but at least ONE of these content sources must be present:
    [files...]          one or more local file paths
    --file name=...     inline content (repeatable)
    --file name=@path   read content from a local file (repeatable)
    stdin (piped)       with --filename to name it
    --interactive       prompts for description + files (TTY only)

## Optional flags
  --description      Gist description (free text)
  --public           Make gist public (default: private)
  --filename         Filename to use when content comes from stdin
                     (default: "untitled.txt")
  --file             One file entry. Format: "name=content" or
                     "name=@/path/to/file". Repeatable.
  --interactive / -i Interactive prompts for description + files
                     (mutually exclusive with stdin / positional args
                     from an agent's perspective — leave off for agents)

## Idempotency
NOT idempotent. Each invocation creates a NEW gist with a fresh id.
Re-running this command creates a duplicate. There is no
"create or update" semantic — to modify an existing gist, use
` + "`" + `gitee gist edit` + "`" + `.

## Atomicity
Atomic. Either the gist (with all its files) is created and an id
is returned, or no change happens on the server. There is no
partial-success state.

## Side effects
- A new gist appears under the authenticated user's account
- If --public: the gist is world-readable and indexable
- Webhooks subscribed to "Gist" events may fire
- The gist has a fresh id; nothing else is touched

## Post-conditions (on success)
- A new gist exists, owned by the authenticated user
- stdout includes the gist id and URL
- exit code 0

## Output (stdout, table mode)
  ✓ Gist created: abc123def
    URL: https://gitee.com/...
    Files: 2

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     token lacks gist scope   →  check token permissions
- 422 Unprocessable: payload rejected          →  check filenames + content
- "no files provided": none of stdin/--file/positional/--interactive
                     supplied content  →  pass at least one source

## Next steps (orchestration hints)
After a successful create, an agent typically wants to:
1. Record the gist id from stdout (do NOT re-list to find it)
2. ` + "`" + `gitee gist view <id>` + "`" + `     — verify what was uploaded
3. ` + "`" + `gitee gist edit <id>` + "`" + `     — tweak description or files
4. ` + "`" + `gitee gist star <id>` + "`" + `     — bookmark it for later
5. ` + "`" + `gitee gist clone <id>` + "`" + `    — bring it back local

## OAuth scope required
projects (write) — or gists (write) on the older token model

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example (agent-friendly, closing the loop)
  # 1. Create a gist from a local script (one-shot)
  OUT=$(gitee gist create --description "Repro for issue #42" \
    --public ./repro.py)
  GIST_ID=$(echo "$OUT" | grep -oE 'Gist created: [a-zA-Z0-9_-]+' \
    | awk '{print $3}')

  # 2. Or create inline (no temp file needed)
  gitee gist create \
    --file main.go="package main
func main() {}" \
    --file README.md="# hello"

  # 3. Verify
  gitee gist view "$GIST_ID"

## See also
- ` + "`" + `gitee gist list` + "`" + `   — discover existing gists
- ` + "`" + `gitee gist view` + "`" + `   — read a gist
- ` + "`" + `gitee gist edit` + "`" + `   — modify a gist (idempotent)
- ` + "`" + `gitee gist clone` + "`" + `  — fetch a gist to local disk
- ` + "`" + `gitee --agent-help` + "`" + ` — overview of all commands with agent help
`
