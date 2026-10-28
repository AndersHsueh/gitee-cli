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

	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"
)

// Options carries the explicit runtime dependencies a list command needs.
// Passing them in (instead of reading package globals) lets cmdkit live in
// the internal layer without importing gitee-cli/cmd — the cmd layer builds
// these from its flags/globals and hands them over.
type Options struct {
	// Client is the API client to use.
	Client *api.Client
	// Timeout is the per-request context timeout.
	Timeout time.Duration
	// Output is the output format: "table" (default) or "json".
	Output string
	// Paginate traverses all pages when true.
	Paginate bool
	// PerPage is the items-per-page for pagination (clamped 1-100).
	PerPage int
	// Limit caps the total items when Paginate is true (0 = no cap).
	Limit int
	// Page is the starting page for a single-page fetch.
	Page int
	// Repo is the global --repo fallback used by the default repo resolver.
	Repo string
}

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

	// Fetch performs the data retrieval. When nil, RunList uses
	// api.FetchListOrAll with the global pagination flags
	// (--paginate/--page/--per-page/--limit). Override it for commands with
	// a non-standard pagination model — e.g. gist list caps per_page with
	// --limit and does a single GET (no --paginate support).
	Fetch func(ctx context.Context, client *api.Client, path string) ([]T, error)

	// Columns is the table header row. Required for table output.
	Columns []string
	// Row renders one item into table cells. Required for table output.
	Row func(item T) []string
	// EmptyText renders the empty-list message. When nil, nothing is
	// printed for an empty result.
	EmptyText func(owner, repo string) string
	// Footer renders a table footer after the rows (e.g. "\nTotal: N
	// label(s)"). When nil, no footer is printed.
	Footer func(count int) string
	// JSON overrides the default indented-JSON output. When nil, RunList
	// uses json.MarshalIndent + fmt.Println. Set this to reuse a shared
	// JSON printer (e.g. handler.PrintJSON) without importing handler here.
	JSON func(items []T) error
	// ErrCtx prefixes non-404 fetch errors (e.g. "listing issues").
	ErrCtx string
}

// RunList executes the shared list skeleton against the supplied spec.
//
// It owns: repo resolution, client construction, timeout, pagination
// (--paginate/--page/--per-page/--limit), json/table output dispatch,
// empty-result messaging, and 404→empty handling. Global behavior is
// identical to the pre-skeleton list commands.
func RunList[T any](opts Options, spec ListSpec[T]) error {
	// 1. Resolve the repository.
	resolve := spec.ResolveOwnerRepo
	if resolve == nil {
		resolve = func() (string, string, error) {
			return utils.ResolveRepo("", opts.Repo, nil)
		}
	}
	owner, repo, err := resolve()
	if err != nil {
		return err
	}

	// 2. Use the injected API client.
	client := opts.Client
	if client == nil {
		return fmt.Errorf("cmdkit: Options.Client is nil")
	}

	// 3. Apply the shared timeout.
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// 4. Fetch. Default is FetchListOrAll (single page or all pages per
	// --paginate; 404 → empty). A command with a non-standard pagination
	// model (e.g. gist list) supplies its own Fetch.
	var items []T
	if spec.Fetch != nil {
		items, err = spec.Fetch(ctx, client, spec.Path(owner, repo))
	} else {
		limit := opts.Limit
		if spec.Limit != nil {
			limit = spec.Limit()
		}
		// The single-page/--paginate limit semantics live in
		// api.FetchListOrAll (one shared implementation for every read
		// path); the skeleton only forwards the values.
		items, err = api.FetchListOrAll[T](ctx, client, spec.Path(owner, repo),
			opts.Paginate, opts.PerPage, limit, opts.Page, spec.ErrCtx)
	}
	if err != nil {
		return err
	}

	// 5. Dispatch output.
	if opts.Output == "json" {
		if spec.JSON != nil {
			return spec.JSON(items)
		}
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
	if err := w.Flush(); err != nil {
		return err
	}
	if spec.Footer != nil {
		fmt.Print(spec.Footer(len(items)))
	}
	return nil
}
