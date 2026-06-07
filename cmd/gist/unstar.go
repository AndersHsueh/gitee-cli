package gist

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/spf13/cobra"
)

var unstarCmd = &cobra.Command{
	Use:     "unstar <id>",
	Short:   "Unstar a gist",
	Example: `  gitee gist unstar <gistID>`,
	Args:    cobra.ExactArgs(1),
	RunE:    runUnstar,
}

func runUnstar(c *cobra.Command, args []string) error {
	gistID := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.GistUnstar(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("unstarring gist: %w", err)
	}

	fmt.Printf("Gist %s unstarred\n", gistID)
	return nil
}
