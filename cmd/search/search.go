package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/config"
	"gitee-cli/internal/utils"
	"github.com/spf13/cobra"
)

var SearchCmd = &cobra.Command{
	Use:   "search <subcommand>",
	Short: "Search for repositories, issues, or users on Gitee",
	Example: `  gitee search repos gitee
  gitee search issues "bug fix"
  gitee search users johndoe`,
}

// newSearchClient creates a client; token is optional for public search endpoints
func newSearchClient() *api.Client {
	token := cmd.GlobalToken
	if token == "" {
		token = config.GetToken(cmd.GlobalHostname)
	}
	return api.NewClient(token, cmd.GlobalAPIURL, cmd.GlobalVerbose, cmd.GlobalTimeout)
}

// ----- repos -----

var reposFlags struct {
	language string
	sort     string
	order    string
	limit    int
}

var searchReposCmd = &cobra.Command{
	Use:   "repos <query>",
	Short: "Search repositories",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearchRepos,
}

func init() {
	f := searchReposCmd.Flags()
	f.StringVar(&reposFlags.language, "language", "", "Filter by programming language")
	f.StringVar(&reposFlags.sort, "sort", "", "Sort by: stars|forks|updated")
	f.StringVar(&reposFlags.order, "order", "desc", "Sort order: asc|desc")
	f.IntVarP(&reposFlags.limit, "limit", "L", 20, "Maximum number of results")

	SearchCmd.AddCommand(searchReposCmd, searchIssuesCmd, searchUsersCmd)
}

func runSearchRepos(c *cobra.Command, args []string) error {
	q := args[0]
	client := newSearchClient()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// Build path
	path := fmt.Sprintf("/search/repositories?q=%s&per_page=%d", q, reposFlags.limit)
	if reposFlags.sort != "" {
		path += "&sort=" + reposFlags.sort
	}
	if reposFlags.order != "" {
		path += "&order=" + reposFlags.order
	}
	if reposFlags.language != "" {
		path += "&language=" + reposFlags.language
	}

	var result []interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		return fmt.Errorf("search repos: %w", err)
	}

	return printSearchResults(result, "repos")
}

// ----- issues -----

var issuesFlags struct {
	state  string
	labels string
	limit  int
}

var searchIssuesCmd = &cobra.Command{
	Use:   "issues <query>",
	Short: "Search issues across repositories",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearchIssues,
}

func init() {
	f := searchIssuesCmd.Flags()
	f.StringVar(&issuesFlags.state, "state", "open", "Filter by state: open|closed|all")
	f.StringVar(&issuesFlags.labels, "labels", "", "Filter by labels")
	f.IntVarP(&issuesFlags.limit, "limit", "L", 20, "Maximum number of results")
}

func runSearchIssues(c *cobra.Command, args []string) error {
	q := args[0]
	client := newSearchClient()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	path := fmt.Sprintf("/search/issues?q=%s&per_page=%d", q, issuesFlags.limit)
	if issuesFlags.state != "" && issuesFlags.state != "all" {
		path += "&state=" + issuesFlags.state
	}
	if issuesFlags.labels != "" {
		path += "&labels=" + issuesFlags.labels
	}

	var result interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		return fmt.Errorf("search issues: %w", err)
	}

	// Convert to array for unified printing
	var items []interface{}
	if arr, ok := result.([]interface{}); ok {
		items = arr
	} else if m, ok := result.(map[string]interface{}); ok {
		// Some API responses are wrapped in an object
		if arr, ok := m["items"].([]interface{}); ok {
			items = arr
		} else {
			items = []interface{}{m}
		}
	}

	return printSearchResults(items, "issues")
}

// ----- users -----

var usersFlags struct {
	limit int
}

var searchUsersCmd = &cobra.Command{
	Use:   "users <query>",
	Short: "Search users",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearchUsers,
}

func init() {
	f := searchUsersCmd.Flags()
	f.IntVarP(&usersFlags.limit, "limit", "L", 20, "Maximum number of results")
}

func runSearchUsers(c *cobra.Command, args []string) error {
	q := args[0]
	client := newSearchClient()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	path := fmt.Sprintf("/search/users?q=%s&per_page=%d", q, usersFlags.limit)

	var result interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		return fmt.Errorf("search users: %w", err)
	}

	// Convert to array for unified printing
	var items []interface{}
	if arr, ok := result.([]interface{}); ok {
		items = arr
	} else if m, ok := result.(map[string]interface{}); ok {
		if arr, ok := m["items"].([]interface{}); ok {
			items = arr
		} else {
			items = []interface{}{m}
		}
	}

	return printSearchResults(items, "users")
}

// printSearchResults prints search results in a unified format
func printSearchResults(result interface{}, searchType string) error {
	var items []interface{}

	// Handle different response formats
	switch v := result.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		if arr, ok := v["items"].([]interface{}); ok {
			items = arr
		} else {
			items = []interface{}{v}
		}
	case nil:
		items = []interface{}{}
	default:
		items = []interface{}{}
	}

	// JSON output
	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	// Table output
	if len(items) == 0 {
		fmt.Println("(no results)")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	switch searchType {
	case "repos":
		fmt.Fprintln(w, "NAME\tLANGUAGE\tSTARS\tFORKS\tDESCRIPTION")
		for _, raw := range items {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := m["full_name"].(string)
			lang, _ := m["language"].(string)
			if lang == "" {
				lang = "-"
			}
			stars := fmt.Sprintf("%v", m["stargazers_count"])
			forks := fmt.Sprintf("%v", m["forks_count"])
			desc, _ := m["description"].(string)
			desc = utils.Truncate(desc, 50)
			if desc == "" {
				desc = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, lang, stars, forks, desc)
		}
	case "issues":
		fmt.Fprintln(w, "REPOSITORY\tNUMBER\tTITLE\tSTATE\tLABELS\tUPDATED")
		for _, raw := range items {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			repo, _ := m["repo"].(string)
			number := fmt.Sprintf("%v", m["number"])
			title, _ := m["title"].(string)
			state, _ := m["state"].(string)
			labelsRaw, _ := m["labels"].([]interface{})
			var labels []string
			for _, l := range labelsRaw {
				if lm, ok := l.(map[string]interface{}); ok {
					if name, ok := lm["name"].(string); ok {
						labels = append(labels, name)
					}
				}
			}
			labelStr := utils.Truncate(strings.Join(labels, ","), 20)
			if labelStr == "" {
				labelStr = "-"
			}
			updated, _ := m["updated_at"].(string)
			updated = utils.TruncateDate(updated)
			title = utils.Truncate(title, 40)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", repo, number, title, state, labelStr, updated)
		}
	case "users":
		fmt.Fprintln(w, "USERNAME\tNAME\tFOLLOWERS\tREPOS\tBIO")
		for _, raw := range items {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			login, _ := m["login"].(string)
			name, _ := m["name"].(string)
			if name == "" {
				name = "-"
			}
			followers := fmt.Sprintf("%v", m["followers"])
			repos := fmt.Sprintf("%v", m["public_repos"])
			bio, _ := m["bio"].(string)
			bio = utils.Truncate(bio, 30)
			if bio == "" {
				bio = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", login, name, followers, repos, bio)
		}
	}

	_ = w.Flush()
	return nil
}
