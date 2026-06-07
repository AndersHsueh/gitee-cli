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
	Args:  cobra.ExactArgs(2),
	RunE:  runRaw,
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
	Args: cobra.ExactArgs(2),
	RunE: runAPI,
}

func init() {
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
