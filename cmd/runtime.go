package cmd

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
	"gitee-cli/internal/config"
	"gitee-cli/internal/domain"

	"github.com/spf13/cobra"
)

// Runtime carries the frozen configuration for a single command
// invocation. It is populated once in PersistentPreRunE from the
// resolved flag → env → config priority, then stored in the cobra
// command context. Commands and handlers receive it via
// RuntimeFrom(ctx) instead of reading the package-level Global*
// variables directly.
//
// The package-level Global* variables are kept for backward
// compatibility during the IK923S migration. They are populated from
// the same flag bindings, but new code SHOULD use RuntimeFrom(ctx).
// Once all call sites are migrated, the globals will be removed.
type Runtime struct {
	Token       string
	Hostname    string
	Output      string
	Verbose     bool
	APIURL      string
	Timeout     int
	Repo        string
	NoColor     bool
	AgentHelp   bool
	Paginate    bool
	Page        int
	PerPage     int
	Limit       int
	NoExtensions bool
}

type runtimeKey struct{}

// SetContext stores rt in ctx so subcommands can retrieve it via RuntimeFrom.
func SetContext(ctx context.Context, rt *Runtime) context.Context {
	return context.WithValue(ctx, runtimeKey{}, rt)
}

// RuntimeFrom returns the frozen Runtime carried in ctx, or nil when the
// caller did not go through PersistentPreRunE (e.g. direct test calls).
func RuntimeFrom(ctx context.Context) *Runtime {
	rt, _ := ctx.Value(runtimeKey{}).(*Runtime)
	return rt
}

// RuntimeFromContext is an alias of RuntimeFrom.
func RuntimeFromContext(ctx context.Context) *Runtime { return RuntimeFrom(ctx) }

// NewRuntime builds a Runtime from the resolved globals and the given
// hostname. This is the single place where the flag → env → config
// priority is captured; callers should use RuntimeFrom(ctx) to retrieve
// an already-frozen Runtime, or call NewRuntime directly in tests.
func NewRuntime(hostname string) *Runtime {
	return &Runtime{
		Token:        GlobalToken,
		Hostname:     hostname,
		Output:       GlobalOutput,
		Verbose:      GlobalVerbose,
		APIURL:       GlobalAPIURL,
		Timeout:      GlobalTimeout,
		Repo:         GlobalRepo,
		NoColor:      NoColor,
		AgentHelp:    GlobalAgentHelp,
		Paginate:     GlobalPaginate,
		Page:         GlobalPage,
		PerPage:      GlobalPerPage,
		Limit:        GlobalLimit,
		NoExtensions: GlobalNoExtensions,
	}
}

// RuntimeFromCmd returns the frozen Runtime stored in the command's
// context, falling back to the package-level globals when no Runtime
// has been stored (e.g. in direct test calls that bypass
// PersistentPreRunE). This is the function that all command RunE / Run
// handlers should call to obtain their configuration.
//
// Usage:
//
//	rt := cmd.RuntimeFromCmd(cobraCmd)
//	client, err := cmd.NewAPIClientFrom(rt)
func RuntimeFromCmd(c *cobra.Command) *Runtime {
	if rt := RuntimeFrom(c.Context()); rt != nil {
		return rt
	}
	return NewRuntime(ResolveHostname())
}

// NewAPIClientFrom builds an authenticated API client from a frozen Runtime.
func NewAPIClientFrom(rt *Runtime) (*api.Client, error) {
	hostname := rt.Hostname
	if hostname == "" {
		hostname = "gitee.com"
	}

	token := rt.Token
	if token == "" {
		token = config.GetToken(hostname)
	}
	if token == "" {
		return nil, fmt.Errorf("not authenticated. Set GITEE_TOKEN or run 'gitee auth login'")
	}

	baseURL := rt.APIURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s/api/v5", hostname)
	}
	return api.NewClient(token, baseURL, rt.Verbose, rt.Timeout), nil
}

// NewCmdkitOptionsFrom builds the explicit runtime dependencies for a list
// command's cmdkit.RunList call from a frozen Runtime. The client is
// passed in so callers can reuse one they already built.
func NewCmdkitOptionsFrom(rt *Runtime, client *api.Client) cmdkit.Options {
	return cmdkit.Options{
		Client:   client,
		Timeout:  time.Duration(rt.Timeout) * time.Second,
		Output:   rt.Output,
		Paginate: rt.Paginate,
		PerPage:  rt.PerPage,
		Limit:    rt.Limit,
		Page:     rt.Page,
		Repo:     rt.Repo,
	}
}

// NewDomainClientFrom builds a domain.Client from a frozen Runtime.
func NewDomainClientFrom(rt *Runtime) (*domain.Client, error) {
	apiClient, err := NewAPIClientFrom(rt)
	if err != nil {
		return nil, err
	}
	return &domain.Client{
		API:     apiClient,
		Timeout: time.Duration(rt.Timeout) * time.Second,
		Output:  rt.Output,
		Repo:    rt.Repo,
	}, nil
}