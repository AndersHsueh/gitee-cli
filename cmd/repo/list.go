package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"
)

var listFlags struct {
	owner      string
	visibility string
	limit      int
	sort       string
	direction  string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories",
	Example: `  gitee repo list
  gitee repo list --owner gitee --visibility public
  gitee repo list --sort stars --direction desc`,
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.StringVar(&listFlags.owner, "owner", "", "Filter by owner username")
	f.StringVar(&listFlags.visibility, "visibility", "all", "Filter by visibility: public|private|internal|all")
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of repositories")
	f.StringVar(&listFlags.sort, "sort", "full_name", "Sort by: created|updated|pushed|full_name|stars")
	f.StringVar(&listFlags.direction, "direction", "asc", "Sort direction: asc|desc")
}

func runList(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	var path string
	if listFlags.owner != "" {
		path = api.ReposList(listFlags.owner)
	} else {
		path = api.ReposList("")
	}
	path += fmt.Sprintf("?visibility=%s&sort=%s&direction=%s&per_page=%d",
		listFlags.visibility, listFlags.sort, listFlags.direction, listFlags.limit)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repos []types.Repository
	if err := client.Get(ctx, path, &repos); err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(repos, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVISIBILITY\tLANGUAGE\tSTARS\tFORKS\tUPDATED")
	for _, r := range repos {
		vis := "public"
		if r.Private {
			vis = "private"
		}
		lang := r.Language
		if lang == "" {
			lang = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%s\n",
			r.FullName, vis, lang, r.StarsCount, r.ForksCount, utils.TruncateDate(r.UpdatedAt))
	}
	return w.Flush()
}
