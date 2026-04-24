package gist

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
	"gitee-cli/pkg/types"
)

var listFlags struct {
	limit int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your gists",
	Example: `  gitee gist list
  gitee gist list --limit 20`,
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of gists")
}

func runList(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s?per_page=%d", api.GistsList(), listFlags.limit)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var gists []types.Gist
	if err := client.Get(ctx, path, &gists); err != nil {
		return fmt.Errorf("listing gists: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(gists, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(gists) == 0 {
		fmt.Println("No gists found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDESCRIPTION\tPUBLIC\tFILES\tUPDATED")
	for _, g := range gists {
		vis := "public"
		if !g.Public {
			vis = "private"
		}
		desc := g.Description
		if desc == "" {
			desc = "-"
		}
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		updated := g.UpdatedAt
		if len(updated) > 10 {
			updated = updated[:10]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			g.ID, desc, vis, len(g.Files), updated)
	}
	return w.Flush()
}
