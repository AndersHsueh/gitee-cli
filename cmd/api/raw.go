package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/spf13/cobra"
)

var rawFlags struct {
	method string
	fields []string
	raw    bool
}

var rawCmd = &cobra.Command{
	Use:   "raw <method> <path>",
	Short: "Make raw API request",
	Args:  cmd.AgentHelpAwareArgs(cobra.ExactArgs(2)),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(apiRawAgentHelp)
		}
		return nil
	},
	RunE: runRaw,
}

var apiCmd = &cobra.Command{
	Use:   "api <method> <path>",
	Short: "Make raw API requests",
	Long: `Make raw HTTP requests to the Gitee API.
Useful for testing or accessing endpoints not yet covered by dedicated commands.`,
	Example: `  gitee api GET /user
  gitee api GET /user/repos?type=owner
  gitee api POST /user/repos --field name=my-repo --field private=true
  gitee api DELETE /repos/owner/repo
  gitee api GET /search/repositories?q=gitee --paginate`,
	Args: cmd.AgentHelpAwareArgs(cobra.ExactArgs(2)),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(apiRawAgentHelp)
		}
		return nil
	},
	RunE: runAPI,
}

func init() {
	// --field and --raw are registered on BOTH apiCmd and rawCmd.
	// cobra does NOT inherit a parent's local flags (Flags()) to its
	// subcommands — only PersistentFlags() propagate down. We want:
	//   gitee api POST /user/repos --field name=x   (apiCmd, canonical)
	//   gitee api raw POST /user/repos --field name=x (rawCmd, legacy)
	// to both work. Both bind to the same package-level rawFlags vars,
	// so runAPI reads them regardless of which command was invoked.
	apiCmd.Flags().StringArrayVar(&rawFlags.fields, "field", []string{}, "Form field (key=value)")
	apiCmd.Flags().BoolVar(&rawFlags.raw, "raw", false, "Output raw response body")
	rawCmd.Flags().StringArrayVar(&rawFlags.fields, "field", []string{}, "Form field (key=value)")
	rawCmd.Flags().BoolVar(&rawFlags.raw, "raw", false, "Output raw response body")

	// Add raw as a subcommand for backwards compatibility
	apiCmd.AddCommand(rawCmd)

	// Register global flags for paginate support
	apiCmd.Flags().BoolP("paginate", "P", false, "Paginate through all results")
	apiCmd.Flags().Int("page", 1, "Page number")
	apiCmd.Flags().Int("per-page", 30, "Results per page (max 100)")
	apiCmd.Flags().Int("limit", 0, "Maximum number of results (0 = no limit)")
}

func runAPI(cobraCmd *cobra.Command, args []string) error {
	method := strings.ToUpper(args[0])
	path := args[1]

	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	if !validMethods[method] {
		return fmt.Errorf("invalid method: %s (valid: GET, POST, PUT, PATCH, DELETE)", method)
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// Parse fields into body map
	var body map[string]interface{}
	if len(rawFlags.fields) > 0 && method != "GET" && method != "DELETE" {
		body = make(map[string]interface{})
		for _, f := range rawFlags.fields {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) == 2 {
				body[parts[0]] = parts[1]
			}
		}
	}

	// Check for paginate flag
	paginate, _ := cobraCmd.Flags().GetBool("paginate")

	if paginate && method == "GET" {
		return runPaginatedGet(client, ctx, path, cobraCmd)
	}

	var result json.RawMessage

	switch method {
	case "GET":
		err = client.Get(ctx, path, &result)
	case "POST":
		err = client.Post(ctx, path, body, &result)
	case "PATCH":
		err = client.Patch(ctx, path, body, &result)
	case "DELETE":
		err = client.Delete(ctx, path)
	case "PUT":
		err = client.Put(ctx, path, body, &result)
	}

	if err != nil {
		return err
	}

	if rawFlags.raw {
		io.WriteString(cobraCmd.OutOrStdout(), string(result))
	} else {
		var out bytes.Buffer
		json.Indent(&out, result, "", "  ")
		out.WriteTo(cobraCmd.OutOrStdout())
	}
	return nil
}

func runPaginatedGet(client *api.Client, ctx context.Context, path string, cobraCmd *cobra.Command) error {
	page, _ := cobraCmd.Flags().GetInt("page")
	perPage, _ := cobraCmd.Flags().GetInt("per-page")
	limit, _ := cobraCmd.Flags().GetInt("limit")

	// Ensure perPage is within valid range
	if perPage > 100 {
		perPage = 100
	}
	if perPage < 1 {
		perPage = 30
	}

	allResults := []interface{}{}
	currentPage := page
	totalCollected := 0

	for {
		// Build paginated URL
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		pagePath := fmt.Sprintf("%s%spage=%d&per_page=%d", path, sep, currentPage, perPage)

		var result json.RawMessage
		if err := client.Get(ctx, pagePath, &result); err != nil {
			return fmt.Errorf("paginated request: %w", err)
		}

		// Try to parse as array
		var items []interface{}
		if err := json.Unmarshal(result, &items); err != nil {
			// Not an array, just output as-is
			var out bytes.Buffer
			json.Indent(&out, result, "", "  ")
			out.WriteTo(cobraCmd.OutOrStdout())
			return nil
		}

		if len(items) == 0 {
			break
		}

		allResults = append(allResults, items...)
		totalCollected += len(items)

		// Check limit
		if limit > 0 && totalCollected >= limit {
			break
		}

		// Check if there are more pages
		if len(items) < perPage {
			break
		}

		currentPage++
	}

	// Output all results
	out, err := json.MarshalIndent(allResults, "", "  ")
	if err != nil {
		return err
	}
	cobraCmd.OutOrStdout().Write(out)
	return nil
}

func runRaw(cobraCmd *cobra.Command, args []string) error {
	// For backwards compatibility, delegate to runAPI
	return runAPI(cobraCmd.Parent(), args)
}

// apiRawAgentHelp is the agent-targeted help for `gitee api raw` and
// the parent `gitee api` command. It is intentionally reachable
// WITHOUT a token: agents use --agent-help as a discovery probe and
// should not be blocked by the auth check in runAPI.
//
// Built via fmt.Sprintf to avoid raw-string-literal nesting limits;
// the markdown body uses double-quoted concatenation everywhere an
// inline code span appears so the final string is a single literal.
const apiRawAgentHelp = `# gitee api raw — Agent Help

## Purpose
Make a raw HTTP request to any Gitee v5 API endpoint. Use this as an
escape hatch when the dedicated verbs do not cover an endpoint yet
(very rare after v0.11.0+), or to script multi-step workflows without
needing a dedicated CLI verb for each call.

The same command is exposed under two names:
  - ` + "`" + `gitee api <method> <path>` + "`" + `      (canonical)
  - ` + "`" + `gitee api raw <method> <path>` + "`" + `  (legacy alias)

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Public read endpoints work
  without a token.
- <method> is one of: GET, POST, PUT, PATCH, DELETE (case-insensitive)
- <path> starts with ` + "`" + `/` + "`" + ` and matches a Gitee v5 endpoint

## Required arguments
  <method>   HTTP method (GET | POST | PUT | PATCH | DELETE)
  <path>     Gitee API path, including leading slash (e.g. ` + "`" + `/user` + "`" + `)

## Required flags
  none — but most write endpoints need form fields via ` + "`" + `--field` + "`" + `

## Optional flags
  --field key=value   Form field, repeatable. Used for POST/PUT/PATCH
                      bodies (ignored for GET and DELETE).
  --raw               Print the response body without JSON indent
  --paginate / -P     Walk all pages of a paginated GET response
  --page N            Page number for single-page mode (default 1)
  --per-page N        Items per page (default 30, max 100)
  --limit N           Total cap across pages when --paginate is set

## Idempotency
Depends on <method>:
  - GET / DELETE: idempotent (re-running yields same result)
  - POST / PUT / PATCH: NOT idempotent (each call creates/updates state)

## Atomicity
A single HTTP request. Either the full response is returned or the
call fails. No partial-success state at the wire level; downstream
side effects on the Gitee side depend on the endpoint.

## Side effects
- Depends entirely on the endpoint called. The CLI itself does not
  mutate any local state (it does not write to ` + "`" + `~/.gitee-cli/` + "`" + `).
- Rate-limit counter advances by 1 (or N pages when --paginate is set).

## Post-conditions (on success)
- Stdout contains the response body (JSON, pretty-printed unless
  --raw is set)
- Exit code is 0

## Common errors
- 401 Unauthorized: token missing or expired  →  gitee auth login
- 403 Forbidden:    token lacks required scope for the endpoint
- 404 Not Found:    endpoint path is wrong
- 422 Unprocessable: payload validation failed (form fields)
- Network timeouts: bump --timeout and retry

## Next steps (orchestration hints)
After ` + "`" + `gitee api raw` + "`" + `, an agent typically wants to:
1. Pipe through jq to extract a field:
   ` + "`" + `gitee api GET /user/repos --paginate | jq -r '.[].full_name'` + "`" + `
2. Compare to a list of dedicated verbs first — if a verb exists for
   this endpoint, prefer it (more flags, structured output, retry).
3. If you discover an endpoint missing a dedicated verb, file an
   issue mentioning "raw needed for <path>".

## OAuth scope required
Depends on endpoint. Public GETs need no scope; private resources
need projects (read or write); admin actions need user_info.

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.

## Example
  # Fetch the authenticated user
  gitee api GET /user

  # Paginate all of alice's repos
  gitee api GET /users/alice/repos --paginate --limit 100

  # POST a new comment (uses --field for the body)
  gitee api POST /repos/owner/repo/issues/42/comments \
    --field body='Thanks for the report!'

## See also
- ` + "`" + `gitee --agent-help` + "`" + `      — overview of all commands
- ` + "`" + `gitee help reference` + "`" + ` — index of dedicated verbs
`
