package org

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

var memberFlags struct {
	org   string
	limit int
}

var memberCmd = &cobra.Command{
	Use:   "members <org>",
	Short: "List organization members",
	Example: `  gitee org members my-org`,
	RunE: runMembers,
}

func init() {
	f := memberCmd.Flags()
	f.StringVar(&memberFlags.org, "org", "", "Organization name")
	f.IntVarP(&memberFlags.limit, "limit", "L", 30, "Maximum number of members")
}

func runMembers(cobraCmd *cobra.Command, args []string) error {
	org := memberFlags.org
	if len(args) > 0 {
		org = args[0]
	}
	if org == "" {
		return fmt.Errorf("organization name is required")
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s?per_page=%d", api.OrgMembers(org), memberFlags.limit)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var members []types.OrgMember
	if err := client.Get(ctx, path, &members); err != nil {
		return fmt.Errorf("listing org members: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(members, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(members) == 0 {
		fmt.Println("No members found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "LOGIN\tNAME\tEMAIL\tSTATE")
	for _, m := range members {
		email := m.Email
		if email == "" {
			email = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.Login, m.Name, email, m.State)
	}
	return w.Flush()
}
