// Package cmdkit provides deep-module skeletons for the command layer.
//
// The gitee-cli list/view commands historically repeated the same
// five-stage boilerplate in every file:
//
//	resolveRepo → cmd.NewAPIClient() → context.WithTimeout(cmd.GlobalTimeout)
//	  → api.FetchListOrAll (pagination) → json/table output dispatch
//
// RunList centralizes that skeleton: the business-specific parts (repo
// resolution, path building, limit source, columns, row rendering, empty
// text) are declared in a ListSpec, and the skeleton supplies everything
// else. Global behavior (--output, --paginate, --limit, 404→empty) is
// implemented once here so it can never drift between modules.
package cmdkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"
)

// ListSpec declares the business-specific parts of a list command.
type ListSpec[T any] struct {
	// ResolveOwnerRepo returns (owner, repo). When nil, RunList uses a
	// default that resolves the global --repo flag (with git-config
	// fallback) via utils.ResolveRepo.
	ResolveOwnerRepo func() (owner, repo string, err error)

	// Path returns the base request path for one list page. It may already
	// carry a query string; page/per_page are appended by the pagination
	// layer. Required.
	Path func(owner, repo string) string

	// Limit returns the total cap for --paginate (0 = no cap). When nil,
	// RunList uses cmd.GlobalLimit.
	Limit func() int

	// Columns is the table header row. Required for table output.
	Columns []string
	// Row renders one item into table cells. Required for table output.
	Row func(item T) []string
	// EmptyText renders the empty-list message. When nil, nothing is
	// printed for an empty result.
	EmptyText func(owner, repo string) string
	// ErrCtx prefixes non-404 fetch errors (e.g. "listing issues").
	ErrCtx string
}

// RunList executes the shared list skeleton against the supplied spec.
//
// It owns: repo resolution, client construction, timeout, pagination
// (--paginate/--page/--per-page/--limit), json/table output dispatch,
// empty-result messaging, and 404→empty handling. Global behavior is
// identical to the pre-skeleton list commands.
func RunList[T any](spec ListSpec[T]) error {
	// 1. Resolve the repository.
	resolve := spec.ResolveOwnerRepo
	if resolve == nil {
		resolve = func() (string, string, error) {
			return utils.ResolveRepo("", cmd.GlobalRepo, nil)
		}
	}
	owner, repo, err := resolve()
	if err != nil {
		return err
	}

	// 2. Build the API client.
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	// 3. Apply the shared timeout.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// 4. Fetch (single page or all pages per --paginate; 404 → empty).
	limit := cmd.GlobalLimit
	if spec.Limit != nil {
		limit = spec.Limit()
	}
	items, err := api.FetchListOrAll[T](ctx, client, spec.Path(owner, repo),
		cmd.GlobalPaginate, cmd.GlobalPerPage, limit, cmd.GlobalPage, spec.ErrCtx)
	if err != nil {
		return err
	}

	// 5. Dispatch output.
	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(items) == 0 {
		if spec.EmptyText != nil {
			fmt.Println(spec.EmptyText(owner, repo))
		}
		return nil
	}

	w := output.NewTableWriter()
	fmt.Fprintln(w, strings.Join(spec.Columns, "\t"))
	for _, it := range items {
		fmt.Fprintln(w, strings.Join(spec.Row(it), "\t"))
	}
	return w.Flush()
}
