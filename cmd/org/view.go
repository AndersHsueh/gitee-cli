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

var viewFlags struct {
	org string
}

var viewCmd = &cobra.Command{
	Use:   "view [<org>]",
	Short: "View organization details",
	Example: `  gitee org view my-org`,
	RunE: runView,
}

func init() {
	f := viewCmd.Flags()
	f.StringVar(&viewFlags.org, "org", "", "Organization name")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	org := viewFlags.org
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

	path := api.OrgGet(org)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var o types.Organization
	if err := client.Get(ctx, path, &o); err != nil {
		return fmt.Errorf("viewing org: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(o, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Login:\t%s\n", o.Login)
	fmt.Fprintf(w, "Name:\t%s\n", o.Name)
	fmt.Fprintf(w, "Description:\t%s\n", o.Description)
	fmt.Fprintf(w, "Email:\t%s\n", o.Email)
	fmt.Fprintf(w, "Blog:\t%s\n", o.Blog)
	fmt.Fprintf(w, "Public Repos:\t%d\n", o.PublicRepos)
	fmt.Fprintf(w, "Followers:\t%d\n", o.Followers)
	fmt.Fprintf(w, "Following:\t%d\n", o.Following)
	fmt.Fprintf(w, "URL:\t%s\n", o.HTMLURL)
	return w.Flush()
}
