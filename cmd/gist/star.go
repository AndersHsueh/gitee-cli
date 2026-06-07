package gist

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/spf13/cobra"
)

var starCmd = &cobra.Command{
	Use:     "star <id>",
	Short:   "Star a gist",
	Example: `  gitee gist star abc123`,
	RunE:    runStar,
}

func runStar(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("gist ID is required")
	}
	gistID := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.GistStar(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Put(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("starring gist: %w", err)
	}

	fmt.Printf("Gist %s starred\n", gistID)
	return nil
}
