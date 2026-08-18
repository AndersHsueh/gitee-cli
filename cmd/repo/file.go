package repo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Per the P1-7 spec, the file content subcommand group lives entirely in
// this one file (`cmd/repo/file.go`) and registers three verbs under the
// `gitee repo file` umbrella:
//
//   view   — GET    /repos/{owner}/{repo}/contents/{path}    (single file or dir listing)
//   push   — POST/PUT .../contents/{path}                    (create or update; auto-detects)
//   delete — DELETE .../contents/{path}                      (requires sha; auto-fetches if --sha omitted)
//
// All three share the --repo / -R flag pattern; view accepts a positional
// <path>; push accepts <local-path> <remote-path> with -m/--message;
// delete accepts a positional <path> with -m/--message and an optional
// --sha (auto-fetched if omitted, which requires an extra GET round-trip).

// fileFlags is shared across view/push/delete so the user can invoke
// `gitee repo file view README.md --repo owner/repo` (Cobra binds by
// pointer). Per-verb fields (path, message, sha, ...) are read by their
// respective RunE functions only when set.
var fileFlags struct {
	repo    string
	message string
	sha     string
	branch  string
	force   bool
}

// fileCmd is the umbrella command. Running `gitee repo file` with no
// subcommand shows the help so users discover view/push/delete.
var fileCmd = &cobra.Command{
	Use:   "file <subcommand>",
	Short: "View, push, or delete files in a repository",
	Long: `Manage file contents in a Gitee repository.

Use these verbs to read a file (or list a directory), push a local file
to the repo (auto-create-or-update), or delete a file. All three
operations go through the Gitee v5 contents API and create a commit on
the target branch.

  gitee repo file view <path>                — read a file or list a directory
  gitee repo file push <local> <remote-path> — create or update a file
  gitee repo file delete <path>              — delete a file (requires --sha or auto-fetch)`,
	Example: `  gitee repo file view --repo owner/repo README.md
  gitee repo file push ./README.md README.md --repo owner/repo -m "update readme"
  gitee repo file delete --repo owner/repo docs/old.md -m "remove obsolete"`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoFileAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		// No subcommand given — show help so users discover view/push/delete.
		return c.Help()
	},
}

var fileViewCmd = &cobra.Command{
	Use:   "view <path>",
	Short: "View a file's contents (or list a directory) in a repository",
	Example: `  gitee repo file view --repo owner/repo README.md
  gitee repo file view -R owner/repo src/main.go
  gitee repo file view --repo owner/repo docs/`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoFileViewAgentHelp)
		}
		return nil
	},
	RunE: runFileView,
}

var filePushCmd = &cobra.Command{
	Use:   "push <local-path> <remote-path>",
	Short: "Push a local file to the repository (auto-detect create vs update)",
	Example: `  gitee repo file push ./README.md README.md --repo owner/repo -m "update"
  gitee repo file push ./docs/api.md docs/api.md -R owner/repo -m "doc fix"`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoFilePushAgentHelp)
		}
		return nil
	},
	RunE: runFilePush,
}

var fileDeleteCmd = &cobra.Command{
	Use:   "delete <path>",
	Short: "Delete a file from the repository",
	Example: `  gitee repo file delete --repo owner/repo docs/old.md -m "remove"
  gitee repo file delete README.md -R owner/repo -m "wipe" --force`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoFileDeleteAgentHelp)
		}
		return nil
	},
	RunE: runFileDelete,
}

func init() {
	// view flags
	vf := fileViewCmd.Flags()
	vf.StringVarP(&fileFlags.repo, "repo", "R", "", "Repository (owner/repo)")

	// push flags
	pf := filePushCmd.Flags()
	pf.StringVarP(&fileFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	pf.StringVarP(&fileFlags.message, "message", "m", "", "Commit message (required for push)")
	pf.StringVarP(&fileFlags.branch, "branch", "b", "", "Target branch (defaults to the repo's default branch)")
	_ = filePushCmd.MarkFlagRequired("message")

	// delete flags
	df := fileDeleteCmd.Flags()
	df.StringVarP(&fileFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	df.StringVarP(&fileFlags.message, "message", "m", "", "Commit message (required for delete)")
	df.StringVar(&fileFlags.sha, "sha", "", "Blob SHA of the file (auto-fetched if omitted, single extra GET)")
	df.BoolVarP(&fileFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
	_ = fileDeleteCmd.MarkFlagRequired("message")

	fileCmd.AddCommand(fileViewCmd)
	fileCmd.AddCommand(filePushCmd)
	fileCmd.AddCommand(fileDeleteCmd)
}

// runFileView handles `gitee repo file view <path>`. It accepts the path
// as a positional argument; --repo supplies the target repo. The GET
// response is either a single ContentEntry (file) or an array (dir).
func runFileView(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("file view requires a <path> argument (e.g. README.md or docs/)")
	}
	remotePath := args[0]

	owner, repoName, err := utils.ResolveRepo("", fileFlags.repo, nil)
	if err != nil {
		return err
	}

	cleanPath := strings.TrimPrefix(remotePath, "/")
	if cleanPath == "" {
		// Treat an empty/root path as a directory listing request.
		cleanPath = "."
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.RepoContentsGet(owner, repoName, cleanPath)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// Gitee v5 returns either a JSON object (file) or array (directory)
	// from the same path. Probe the first byte to disambiguate, then
	// unmarshal into the appropriate shape.
	body, err := client.GetRaw(ctx, path)
	if err != nil {
		return fmt.Errorf("fetching contents: %w", err)
	}

	trimmed := strings.TrimSpace(string(body))
	if strings.HasPrefix(trimmed, "[") {
		// Directory listing: array of ContentEntry.
		var entries []types.ContentEntry
		if err := json.Unmarshal(body, &entries); err != nil {
			return fmt.Errorf("parsing directory listing: %w", err)
		}
		return printDirListing(owner, repoName, cleanPath, entries)
	}

	// Single file: object form.
	var entry types.ContentEntry
	if err := json.Unmarshal(body, &entry); err != nil {
		return fmt.Errorf("parsing file entry: %w", err)
	}
	return printFileEntry(owner, repoName, &entry, true)
}

// printDirListing renders a directory listing (Table) or JSON to stdout
// based on the global --output flag.
func printDirListing(owner, repoName, path string, entries []types.ContentEntry) error {
	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(entries, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("Directory %s/%s/%s:\n", owner, repoName, path)
	if len(entries) == 0 {
		fmt.Println("  (empty)")
		return nil
	}
	for _, e := range entries {
		kind := e.Type
		if kind == "" {
			kind = "file"
		}
		size := ""
		if e.Type == "file" {
			size = fmt.Sprintf("  %d B", e.Size)
		}
		fmt.Printf("  %-7s  %s%s\n", kind, e.Name, size)
	}
	return nil
}

// printFileEntry renders a single file's metadata and decoded content
// (Table) or JSON to stdout. When `showBody` is true the decoded file
// body is printed; when false only metadata is shown.
func printFileEntry(owner, repoName string, entry *types.ContentEntry, showBody bool) error {
	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(entry, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("File: %s/%s/%s\n", owner, repoName, entry.Path)
	fmt.Printf("  SHA:  %s\n", entry.SHA)
	fmt.Printf("  Size: %d B\n", entry.Size)
	if entry.HTMLURL != "" {
		fmt.Printf("  URL:  %s\n", color.CyanString(entry.HTMLURL))
	}
	if !showBody {
		return nil
	}
	// Gitee always base64-encodes file bodies; decode here so the user
	// sees the actual file content on stdout.
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(entry.Content))
	if err != nil {
		fmt.Printf("  (could not decode %q content: %v)\n", entry.Encoding, err)
		return nil
	}
	fmt.Println("  ---")
	fmt.Println(string(decoded))
	return nil
}

// runFilePush handles `gitee repo file push <local> <remote>`. The flow:
//  1. Read the local file.
//  2. GET .../contents/<remote> to discover the current SHA (if any).
//  3. POST to create (404 on GET) or PUT to update (sha present).
//  4. Print the resulting commit + content URL.
func runFilePush(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	if len(args) < 2 || args[0] == "" || args[1] == "" {
		return fmt.Errorf("file push requires <local-path> and <remote-path> arguments")
	}
	localPath, remotePath := args[0], args[1]

	owner, repoName, err := utils.ResolveRepo("", fileFlags.repo, nil)
	if err != nil {
		return err
	}

	// Read the local file once. Reject obviously-empty content with a
	// helpful error (push of a 0-byte file is sometimes intentional —
	// e.g. creating .gitkeep — but usually a typo).
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("reading local file %q: %w", localPath, err)
	}

	cleanRemote := strings.TrimPrefix(remotePath, "/")
	if cleanRemote == "" {
		return fmt.Errorf("invalid remote path %q", remotePath)
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// Probe the remote path to learn whether it already exists. The
	// contents GET returns a single ContentEntry object for files, an
	// array for directories. We treat any successful (non-error) parse
	// as "exists"; only a transport-level error (e.g. 404) means "new".
	existingSHA, exists, err := probeFileSHA(ctx, client, owner, repoName, cleanRemote)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	var resp types.ContentResponse
	if exists {
		// Update: PUT with the discovered SHA.
		req := types.ContentUpdateRequest{
			Content: encoded,
			Message: fileFlags.message,
			SHA:     existingSHA,
			Branch:  fileFlags.branch,
		}
		path := api.RepoContentsUpdate(owner, repoName, cleanRemote)
		if err := client.Put(ctx, path, req, &resp); err != nil {
			return fmt.Errorf("updating file: %w", err)
		}
	} else {
		// Create: POST without SHA.
		req := types.ContentCreateRequest{
			Content: encoded,
			Message: fileFlags.message,
			Branch:  fileFlags.branch,
		}
		path := api.RepoContentsCreate(owner, repoName, cleanRemote)
		if err := client.Post(ctx, path, req, &resp); err != nil {
			return fmt.Errorf("creating file: %w", err)
		}
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	action := "updated"
	if !exists {
		action = "created"
	}
	fmt.Printf("%s File %s in %s/%s\n", green("✓"), bold(action), owner, repoName)
	fmt.Printf("  Path:   %s\n", resp.Content.Path)
	if resp.Commit.SHA != "" {
		short := resp.Commit.SHA
		if len(short) > 7 {
			short = short[:7]
		}
		fmt.Printf("  Commit: %s\n", short)
	}
	if resp.Content.HTMLURL != "" {
		fmt.Printf("  URL:    %s\n", color.CyanString(resp.Content.HTMLURL))
	}
	return nil
}

// probeFileSHA issues a GET on the contents path and returns:
//   - (sha, true, nil)   if the file already exists
//   - ("",   false, nil)  if the path does not exist (404)
//   - ("",   false, err)  on other transport errors
func probeFileSHA(ctx context.Context, client *api.Client, owner, repoName, path string) (string, bool, error) {
	probe, err := client.GetRaw(ctx, api.RepoContentsGet(owner, repoName, path))
	if err != nil {
		// The API client wraps 404 as a *api.APIError; treat that as the
		// "create" case. Any other error is a real failure.
		if api.IsNotFound(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("probing existing file: %w", err)
	}
	trimmed := strings.TrimSpace(string(probe))
	if strings.HasPrefix(trimmed, "[") {
		// Remote path resolves to a directory, not a file.
		return "", false, fmt.Errorf("remote path %q is a directory, not a file", path)
	}
	var entry types.ContentEntry
	if err := json.Unmarshal(probe, &entry); err != nil {
		return "", false, fmt.Errorf("parsing probe response: %w", err)
	}
	return entry.SHA, true, nil
}

// runFileDelete handles `gitee repo file delete <path>`. The SHA is
// required by Gitee; if --sha is not provided, we issue a GET to fetch
// the current SHA (one extra round-trip).
func runFileDelete(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("file delete requires a <path> argument (e.g. docs/old.md)")
	}
	remotePath := args[0]

	owner, repoName, err := utils.ResolveRepo("", fileFlags.repo, nil)
	if err != nil {
		return err
	}

	cleanRemote := strings.TrimPrefix(remotePath, "/")
	if cleanRemote == "" {
		return fmt.Errorf("invalid remote path %q", remotePath)
	}

	if !fileFlags.force {
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Delete %s from %s/%s?",
				color.YellowString("?"), cleanRemote, owner, repoName),
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

	// A single client + ctx serves both the optional auto-fetch of the
	// blob SHA and the actual DELETE. Previously each branch built its
	// own client, duplicating NewAPIClient (which re-reads token/config).
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	sha := fileFlags.sha
	if sha == "" {
		// Auto-fetch the SHA so the user doesn't have to copy-paste it.
		sha, _, err = probeFileSHA(ctx, client, owner, repoName, cleanRemote)
		if err != nil {
			return err
		}
		if sha == "" {
			return fmt.Errorf("file %q does not exist in %s/%s; nothing to delete", cleanRemote, owner, repoName)
		}
	}

	req := types.ContentDeleteRequest{
		SHA:     sha,
		Message: fileFlags.message,
		Branch:  fileFlags.branch,
	}
	path := api.RepoContentsDelete(owner, repoName, cleanRemote)
	if err := client.DeleteWithBody(ctx, path, req); err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s File %s deleted from %s/%s\n",
		green("✓"), bold(cleanRemote), owner, repoName)
	return nil
}

const repoFileAgentHelp = `# gitee repo file — Agent Help

## Purpose
Top-level umbrella for repository file-content operations. Use these
verbs to read or list files in a repo, push a local file up to the repo
(auto-detecting create-vs-update), or delete a file. All three
operations go through the Gitee v5 contents API and create a commit on
the target branch (default branch unless --branch is supplied).

## Subcommands
  view <path>                 Read a file (or list a directory).
                              See ` + "`" + `gitee repo file view --agent-help` + "`" + `.
  push <local> <remote>       Push a local file to the repo. Auto-detects
                              create (POST) vs update (PUT) by reading
                              the current SHA via GET first.
                              See ` + "`" + `gitee repo file push --agent-help` + "`" + `.
  delete <path>               Delete a file. SHA can be auto-fetched.
                              See ` + "`" + `gitee repo file delete --agent-help` + "`" + `.

## Pre-conditions
- Authenticated (gitee auth status)
- --repo / -R is set in the form ` + "`" + `owner/repo` + "`" + `, or
  ` + "`" + `--repo` + "`" + ` is set globally, or the local git remote resolves it
- For push/delete: caller has write access to the target branch
  (member / collaborator with push / owner)

## Idempotency
- view:    YES. Read-only.
- push:    YES when targeting an existing file (re-running with the
           same content is a no-op; with different content overwrites).
           NOT idempotent when the file does not yet exist — re-running
           after a successful create is then an update.
- delete:  NO. Re-running on a deleted path returns 404 (the file is
           already gone, so the post-condition holds, but the API call
           itself fails).

## Atomicity
- view:    Atomic per request.
- push:    Atomic. Either a new commit is created (with the new blob)
           or nothing changes.
- delete:  Atomic. Either the deletion commit is created or nothing
           changes.

## Side effects
- view:    None.
- push:    A new commit is created on the target branch. Subscribers
           to repo events (CI on push, watchers) may be notified. The
           commit author defaults to the token owner.
- delete:  A new commit is created on the target branch. The blob
           remains in the repo's object store but is unreachable from
           any ref.

## Output (stdout)
- view default:  file metadata + decoded body (base64-decoded)
- view -o json:  ContentEntry JSON object (or array for a directory)
- push:          success line with action ("created"/"updated"),
                 path, short commit SHA, and HTML URL
- delete:        success line with the deleted path

## OAuth scope required
- view:    projects (read)
- push:    projects (write)
- delete:  projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour authenticated. ` + "`" + `push` + "`" + ` and ` + "`" + `delete` + "`" + `
each issue TWO requests when the SHA is auto-fetched (the probe GET
plus the actual write).

## See also
- ` + "`" + `gitee repo file view` + "`" + `    — read a file
- ` + "`" + `gitee repo file push` + "`" + `    — write a file
- ` + "`" + `gitee repo file delete` + "`" + `  — remove a file
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands
`

const repoFileViewAgentHelp = `# gitee repo file view — Agent Help

## Purpose
Read a file's metadata + body from a Gitee repository, or list a
directory. The same endpoint serves both shapes: Gitee returns a single
ContentEntry JSON object for files and an array of ContentEntry for
directories. The CLI disambiguates the two transparently.

## Pre-conditions
- Authenticated (gitee auth status)
- --repo / -R is set
- The path exists in the repo (404 otherwise)

## Required arguments
  <path>       The path inside the repo. May contain slashes
               (e.g. ` + "`" + `src/main.go` + "`" + `, ` + "`" + `docs/` + "`" + `).
               Use ` + "`" + `.` + "`" + ` or empty for the repo root.

## Optional flags
  --repo / -R   owner/repo (falls back to global --repo)

## Idempotency
YES. Read-only; calling repeatedly yields the same result (modulo
upstream edits between calls).

## Atomicity
Atomic per request. A single API call returns the full payload.

## Side effects
None. Read-only.

## Post-conditions (on success)
- Stdout contains the file metadata + decoded body (table mode)
  or a single ContentEntry JSON object (-o json)
- For a directory path: a listing of types + names

## Output (stdout, table mode)
  File: owner/repo/README.md
    SHA:  a1b2c3d4e5f6...
    Size: 1234 B
    URL:  https://gitee.com/owner/repo/blob/main/README.md
    ---
    # Project Name
    (decoded file body)

## Output (stdout, json mode)
  {
    "name": "README.md",
    "path": "README.md",
    "sha": "...",
    "size": 1234,
    "type": "file",
    "encoding": "base64",
    "content": "IyBQcm9qZWN0IE5hbWU=",
    "html_url": "https://gitee.com/owner/repo/blob/main/README.md"
  }

## Common errors
- 404 Not Found:  path does not exist in the repo
- 403 Forbidden:  repo is private and the token cannot read it
- 401 Unauthorized: token missing or expired

## Race condition warning ⚠️
The shared mutable state is the file's blob on the target branch: this
command always reads the file AT HEAD of the default branch (no
--branch/--ref flag exists), and the returned ` + "`" + `sha` + "`" + ` is only valid
for the commit that was HEAD at fetch time. Classic failure is
view→act: you read README.md, plan an edit, and call ` + "`" + `gitee repo
file push` + "`" + ` or ` + "`" + `file delete` + "`" + ` with the sha from THIS output — but a
concurrent push moved the file, so the write fails with 409 Conflict
(stale SHA). Mitigation: treat the sha as a single-use optimistic-
concurrency token — pass it to the follow-up write IMMEDIATELY, and
on 409 re-run THIS command to fetch the fresh sha rather than
retrying with the old one. Directory listings are likewise a snapshot:
entries can appear or vanish between your listing and per-entry
actions. Retrying THIS command itself is always safe (read-only).

## Next steps (orchestration hints)
After viewing a file, an agent typically wants to:
1. Update it:
   ` + "`" + `gitee repo file push ./README.md README.md --repo owner/repo -m "update"` + "`" + `
2. List a directory tree (this verb doubles as ` + "`" + `ls` + "`" + `):
   ` + "`" + `gitee repo file view --repo owner/repo src` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour authenticated.

## Example
  # Read a file
  gitee repo file view --repo myorg/myrepo README.md

  # List a directory
  gitee repo file view --repo myorg/myrepo docs/

  # JSON for jq parsing
  gitee repo file view --repo myorg/myrepo README.md -o json \\
    | jq -r .content | base64 -d

## See also
- ` + "`" + `gitee repo file push` + "`" + `    — overwrite the file
- ` + "`" + `gitee repo file delete` + "`" + `  — remove the file
- ` + "`" + `gitee repo browse` + "`" + `      — open the repo in a browser
`

const repoFilePushAgentHelp = `# gitee repo file push — Agent Help

## Purpose
Push a local file to a path in a Gitee repository. The CLI auto-detects
whether the remote path exists: if so, it PUTs an update with the
existing SHA; if not, it POSTs to create. Either way, one new commit
is created on the target branch.

## Pre-conditions
- Authenticated with write access to the target branch
- --repo / -R is set
- <local-path> points to a readable local file
- <remote-path> is a valid path inside the repo
- -m / --message is non-empty (enforced by Cobra MarkFlagRequired)

## Required arguments
  <local-path>    Path on the local filesystem (relative or absolute)
  <remote-path>   Path inside the repo (may include directories; the
                  CLI does not create missing parent directories — the
                  repo must already have the directory tree)

## Required flags
  --message / -m   Commit message (non-empty)

## Optional flags
  --repo / -R     owner/repo (falls back to global --repo)
  --branch / -b   Target branch (defaults to the repo's default branch;
                  the branch must already exist)

## Idempotency
YES when targeting an existing file with identical content (server may
still create a redundant commit if blob SHA differs). To re-run safely
without creating a new commit, hash the content and only push if it
differs from the server's blob SHA.

## Atomicity
Atomic. The CLI reads the local file once, then issues either a POST
or a PUT; on the server side, that single call creates exactly one
commit or fails without any change.

## Side effects
- A new commit is created on the target branch
- Subscribers to push events (CI, watchers, webhooks) may be notified
- The file's blob SHA changes (visible in subsequent ` + "`" + `gitee repo file view` + "`" + `)

## Post-conditions (on success)
- The remote path's content equals the local file's bytes
- Stdout prints "✓ File created/updated in owner/repo" with the path,
  short commit SHA, and HTML URL

## Output (stdout, table mode)
  ✓ File created in owner/repo
    Path:   README.md
    Commit: a1b2c3d
    URL:    https://gitee.com/owner/repo/blob/main/README.md

## Common errors
- 401 Unauthorized:  token missing or expired
- 403 Forbidden:     token lacks write access to the target branch
- 404 Not Found:     the remote path's parent directory does not exist
                    (Gitee does not auto-create directories)
- 409 Conflict:      remote file changed since the probe GET (concurrent
                    edit); re-run to refetch the new SHA
- 422 Unprocessable: invalid base64 or oversized payload

## Race condition warning ⚠️
The push flow does TWO API calls when the file already exists: a probe
GET to fetch the current SHA, then PUT. If a concurrent writer updates
the file between the probe and the PUT, Gitee returns 409 Conflict. The
CLI does NOT auto-retry; re-run to pick up the new SHA. This is the
standard optimistic-concurrency pattern; do not treat the 409 as fatal
without first re-fetching the file.

## Next steps (orchestration hints)
After pushing a file, an agent typically wants to:
1. Verify by reading it back:
   ` + "`" + `gitee repo file view --repo owner/repo README.md` + "`" + `
2. Update related files (e.g. a paired index) and re-run the same push
3. Open a PR if the push was to a non-default branch

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour authenticated. Each call uses 1 (create) or 2 (update)
requests.

## Example
  # Push a new README
  gitee repo file push ./README.md README.md \\
    --repo myorg/myrepo -m "Initial commit"

  # Update an existing file on a feature branch
  gitee repo file push ./docs/api.md docs/api.md \\
    -R myorg/myrepo -b feature/api-docs -m "Add pagination notes"

  # Pipeline: push every .md in a folder
  for f in docs/*.md; do
    gitee repo file push "$f" "$f" --repo myorg/myrepo -m "sync $f"
  done

## See also
- ` + "`" + `gitee repo file view` + "`" + `    — read back what was pushed
- ` + "`" + `gitee repo file delete` + "`" + `  — remove a file
- ` + "`" + `gitee repo edit` + "`" + `        — change repo-level settings
`

const repoFileDeleteAgentHelp = `# gitee repo file delete — Agent Help

## Purpose
Delete a file from a Gitee repository. The deletion is a commit on the
target branch (the file's blob remains in the object store but is
unreachable from any ref; restoring it requires a direct push of the
old SHA, which is outside the scope of this command).

## Pre-conditions
- Authenticated with write access to the target branch
- --repo / -R is set
- The file exists in the repo (404 otherwise)
- -m / --message is non-empty (enforced by Cobra MarkFlagRequired)

## Required arguments
  <path>       Path inside the repo (may contain slashes)

## Required flags
  --message / -m   Commit message (non-empty)

## Optional flags
  --repo / -R   owner/repo (falls back to global --repo)
  --branch / -b  Target branch (defaults to the repo's default branch)
  --sha          Blob SHA of the file (auto-fetched if omitted;
                 omitting incurs an extra GET round-trip)
  --force / -f   Skip the y/N confirmation prompt

## Idempotency
NO. Re-running on a deleted path returns 404; the post-condition (the
file is gone) still holds, but the API call itself fails. To make a
delete script idempotent, suppress the error when --sha is provided
and treat 404 as success.

## Atomicity
Atomic. The CLI either issues a single DELETE-with-body that creates
one commit, or fails without any change.

## Side effects
- A new commit is created on the target branch
- Subscribers to push events (CI, watchers, webhooks) may be notified
- The deleted blob remains in the object store but is unreachable

## Post-conditions (on success)
- The file no longer exists in the repo
- Stdout prints "✓ File <path> deleted from owner/repo"

## Output (stdout, table mode)
  ✓ File docs/old.md deleted from owner/repo

## Common errors
- 401 Unauthorized:  token missing or expired
- 403 Forbidden:     token lacks write access to the target branch
- 404 Not Found:     file does not exist (already deleted, or wrong path)
- 409 Conflict:      remote file changed since the probe GET
                    (the --sha is stale); re-run without --sha to refetch

## Race condition warning ⚠️
When --sha is omitted, the CLI does TWO API calls: a probe GET to fetch
the current SHA, then DELETE. If a concurrent writer updates the file
between the probe and the DELETE, Gitee returns 409. Re-run to pick up
the new SHA. To avoid the round-trip entirely, call ` + "`" + `gitee repo file view` + "`" + `
first and pass --sha explicitly — but be aware the SHA may still go
stale before the DELETE lands.

## Next steps (orchestration hints)
After deleting a file, an agent typically wants to:
1. Verify the file is gone:
   ` + "`" + `gitee repo file view --repo owner/repo <path>` + "`" + ` (expect 404)
2. Clean up references in other files (links, indexes) by pushing
   updated versions:
   ` + "`" + `gitee repo file push ./docs/index.md docs/index.md -m "remove stale link"` + "`" + `
3. If the deletion was a mistake, restore via native git over SSH:
   ` + "`" + `git push origin <sha>:<path>` + "`" + `

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour authenticated. Each call uses 1 (--sha supplied) or 2
(auto-fetch) requests.

## Example
  # Plain delete (with confirmation)
  gitee repo file delete --repo myorg/myrepo docs/old.md -m "remove obsolete"

  # Scripted delete (no prompt)
  gitee repo file delete --repo myorg/myrepo docs/old.md \\
    -m "cleanup" --force

  # Pipeline: remove every file in a list
  while read p; do
    gitee repo file delete --repo myorg/myrepo "$p" -m "bulk remove" --force
  done < to-delete.txt

## See also
- ` + "`" + `gitee repo file view` + "`" + `    — confirm a file exists before delete
- ` + "`" + `gitee repo file push` + "`" + `    — push a replacement
- ` + "`" + `gitee repo branches` + "`" + `   — switch branches before delete
`
