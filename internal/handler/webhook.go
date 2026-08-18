// Package handler — webhook subpackage logic.
//
// Webhook handlers follow the same shape as label handlers (per-repo, list/
// view/create/edit/delete) plus two extra verbs that the label module does
// not have: edit (PATCH) and test (POST /tests). Like labels, all writes
// require repo admin rights.
package handler

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
)

// WebhookResolveRepo resolves --repo flag / global --repo into (owner, repo).
// Delegates to utils.ResolveRepo so parsing and the git-config fallback are
// shared across all command groups (label, release). The local flag wins;
// the global --repo is used only when the local flag is empty.
func WebhookResolveRepo(repo string) (string, string, error) {
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	return utils.ResolveRepo("", repo, nil)
}

// WebhookList runs the webhook-list operation.
func WebhookList(repo string, limit int) error {
	return cmdkit.RunList[types.Webhook](cmdkit.ListSpec[types.Webhook]{
		ResolveOwnerRepo: func() (string, string, error) {
			return WebhookResolveRepo(repo)
		},
		Path: func(owner, repoName string) string {
			path := api.WebhooksList(owner, repoName)
			q := url.Values{}
			q.Set("per_page", fmt.Sprintf("%d", limit))
			return path + "?" + q.Encode()
		},
		// webhook list caps per_page with --limit and does a single GET — no
		// --paginate support.
		Fetch: func(ctx context.Context, client *api.Client, path string) ([]types.Webhook, error) {
			var hooks []types.Webhook
			if err := client.Get(ctx, path, &hooks); err != nil {
				return nil, fmt.Errorf("listing webhooks: %w", err)
			}
			return hooks, nil
		},
		Columns: []string{"ID", "URL", "EVENTS", "CREATED"},
		Row: func(h types.Webhook) []string {
			return []string{
				strconv.FormatInt(h.ID, 10),
				h.URL,
				summarizeEvents(h),
				h.CreatedAt.Format("2006-01-02"),
			}
		},
		EmptyText: func(o, r string) string {
			return fmt.Sprintf("No webhooks configured in %s/%s", o, r)
		},
		Footer: func(count int) string {
			return fmt.Sprintf("\nTotal: %d webhook(s)\n", count)
		},
		JSON: func(items []types.Webhook) error {
			return PrintJSON(items)
		},
	})
}

// summarizeEvents turns the 5 boolean event flags into a short comma-joined
// string for the list view. Empty means "no events subscribed" (an unusual
// state — usually Gitee refuses to create a hook with no events).
func summarizeEvents(h types.Webhook) string {
	var names []string
	if h.PushEvents {
		names = append(names, "push")
	}
	if h.TagPushEvents {
		names = append(names, "tag_push")
	}
	if h.IssuesEvents {
		names = append(names, "issues")
	}
	if h.NoteEvents {
		names = append(names, "note")
	}
	if h.MergeRequestEvents {
		names = append(names, "merge_request")
	}
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ",")
}

// WebhookView runs the webhook-view operation.
func WebhookView(repo string, id int64) error {
	if id <= 0 {
		return fmt.Errorf("webhook id is required (positional or --id)")
	}
	owner, repoName, err := WebhookResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var hook types.Webhook
	if err := client.Get(ctx, api.WebhookGet(owner, repoName, id), &hook); err != nil {
		if api.IsNotFound(err) {
			return fmt.Errorf("webhook %d not found in %s/%s", id, owner, repoName)
		}
		return fmt.Errorf("getting webhook: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return PrintJSON(hook)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Webhook Details (id=%s)\n", bold("─"), bold(fmt.Sprintf("%d", hook.ID)))
	fmt.Printf("  URL:      %s\n", green(hook.URL))
	if hook.Title != "" {
		fmt.Printf("  Title:    %s\n", hook.Title)
	}
	fmt.Printf("  Events:   %s\n", yellow(summarizeEvents(hook)))
	fmt.Printf("  Created:  %s\n", hook.CreatedAt.Format(time.RFC3339))
	if hook.Password != "" {
		fmt.Printf("  Password: %s\n", yellow("(set, value hidden)"))
	}
	if hook.Result != "" {
		fmt.Printf("  Last:     %s (HTTP %d)\n", hook.Result, hook.ResultCode)
	}
	fmt.Println()
	return nil
}

// WebhookCreate runs the webhook-create operation.
func WebhookCreate(repo string, req types.WebhookCreateRequest) error {
	if req.URL == "" {
		return fmt.Errorf("--url is required")
	}
	owner, repoName, err := WebhookResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var hook types.Webhook
	if err := client.Post(ctx, api.WebhookCreate(owner, repoName), req, &hook); err != nil {
		return fmt.Errorf("creating webhook: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Webhook created\n", green("✓"))
	fmt.Printf("  ID:    %s\n", bold(fmt.Sprintf("%d", hook.ID)))
	fmt.Printf("  URL:   %s\n", green(hook.URL))
	fmt.Printf("  Events:   %s\n", summarizeEvents(hook))
	fmt.Println()
	return nil
}

// WebhookEdit runs the webhook-update operation (Gitee API uses PATCH).
// Requires URL (per Gitee spec) plus any of the optional fields.
func WebhookEdit(repo string, id int64, req types.WebhookEditRequest) error {
	if id <= 0 {
		return fmt.Errorf("webhook id is required (positional or --id)")
	}
	if req.URL == "" {
		return fmt.Errorf("--url is required (Gitee API: PATCH payload must include the current URL)")
	}
	owner, repoName, err := WebhookResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var hook types.Webhook
	if err := client.Patch(ctx, api.WebhookEdit(owner, repoName, id), req, &hook); err != nil {
		return fmt.Errorf("updating webhook: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Webhook updated (id=%s)\n", green("✓"), bold(fmt.Sprintf("%d", hook.ID)))
	fmt.Printf("  URL:   %s\n", green(hook.URL))
	fmt.Printf("  Events:   %s\n", summarizeEvents(hook))
	fmt.Println()
	return nil
}

// WebhookDelete runs the webhook-delete operation. confirm matches the
// label-delete convention: pass "y" (or set --force) to skip the prompt.
func WebhookDelete(repo string, id int64, confirm string) error {
	if id <= 0 {
		return fmt.Errorf("webhook id is required (positional or --id)")
	}
	owner, repoName, err := WebhookResolveRepo(repo)
	if err != nil {
		return err
	}

	if confirm != "y" && confirm != "Y" && confirm != "yes" {
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Delete webhook id=%d from %s/%s?", color.YellowString("?"), id, owner, repoName),
			false,
		)
		if err != nil {
			return err
		}
		if !yes {
			fmt.Println("Cancelled")
			return nil
		}
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, api.WebhookDelete(owner, repoName, id)); err != nil {
		return fmt.Errorf("deleting webhook: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n%s Webhook id=%d deleted\n", green("✓"), id)
	return nil
}

// WebhookTest fires a test ping (Gitee POSTs to the delivery URL).
// Returns nil on HTTP 204; non-204 from Gitee surfaces as an error.
func WebhookTest(repo string, id int64) error {
	if id <= 0 {
		return fmt.Errorf("webhook id is required (positional or --id)")
	}
	owner, repoName, err := WebhookResolveRepo(repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Post(ctx, api.WebhookTest(owner, repoName, id), nil, nil); err != nil {
		return fmt.Errorf("testing webhook: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n%s Test ping sent to webhook id=%d (delivery result is not inline — inspect with `gitee webhook view %d`)\n",
		green("✓"), id, id)
	return nil
}
