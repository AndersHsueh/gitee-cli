package domain

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
)

// NotificationList runs the notification-list operation.
//
// unreadFilter:
//
//	""     → all threads
//	"true" → only unread
//	"false"→ only read
func (c *Client) NotificationList(unreadFilter string, limit int) error {
	return cmdkit.RunList[types.NotificationThread](c.opts(), cmdkit.ListSpec[types.NotificationThread]{
		// notification list is user-scoped — no target repo.
		ResolveOwnerRepo: func() (string, string, error) {
			return "", "", nil
		},
		Path: func(_, _ string) string {
			path := api.NotificationsList()
			q := url.Values{}
			if unreadFilter == "true" || unreadFilter == "false" {
				q.Set("unread", unreadFilter)
			}
			if limit > 0 {
				q.Set("per_page", strconv.Itoa(limit))
			}
			if encoded := q.Encode(); encoded != "" {
				path = path + "?" + encoded
			}
			return path
		},
		// notification list caps per_page with --limit and does a single GET
		// — no --paginate support.
		Fetch: func(ctx context.Context, client *api.Client, path string) ([]types.NotificationThread, error) {
			var threads []types.NotificationThread
			if err := client.Get(ctx, path, &threads); err != nil {
				return nil, fmt.Errorf("listing notifications: %w", err)
			}
			return threads, nil
		},
		Columns: []string{"ID", "REPO", "TYPE", "SUBJECT", "WHEN"},
		Row: func(t types.NotificationThread) []string {
			marker := " "
			if t.Unread {
				marker = "*"
			}
			subject := utils.Truncate(t.Subject.Title, 50)
			when := t.UpdatedAt.Format("2006-01-02 15:04")
			repoName := t.Repository.FullName
			if repoName == "" {
				repoName = "-"
			}
			return []string{marker + t.ID, repoName, t.Subject.Type, subject, when}
		},
		EmptyText: func(_, _ string) string {
			if unreadFilter == "true" {
				return "No unread notifications. You're all caught up."
			}
			return "No notifications."
		},
		Footer: func(count int) string {
			return fmt.Sprintf("\nTotal: %d thread(s) (* = unread)\n", count)
		},
		JSON: func(items []types.NotificationThread) error {
			return output.PrintJSON(items)
		},
	})
}

// NotificationCount runs the notification-count operation and prints
// just the unread count. In JSON mode it prints the full count object.
func (c *Client) NotificationCount() error {
	ctx, cancel := c.ctx()
	defer cancel()

	var resp types.NotificationUnreadCount
	if err := c.API.Get(ctx, api.NotificationsCount(), &resp); err != nil {
		return fmt.Errorf("getting notification count: %w", err)
	}

	if c.IsJSON() {
		return output.PrintJSON(resp)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	marker := green("0")
	if resp.Count > 0 {
		marker = yellow(strconv.Itoa(resp.Count))
	}
	fmt.Printf("Unread notifications: %s\n", marker)
	return nil
}

// NotificationRead marks threads as read.
//
//	all == true   → mark ALL threads as read (PUT /notifications/threads)
//	all == false  → mark a single thread id as read
//	                (PUT /notifications/threads/{id})
//
// markUnread, when true, flips state back to UNREAD instead. This
// satisfies the inverse operation; useful for agents that want to
// re-surface a thread.
func (c *Client) NotificationRead(threadID string, all bool, markUnread bool, force bool) error {
	if !all && threadID == "" {
		return fmt.Errorf("notification id is required (positional argument or use --all)")
	}

	if all && threadID != "" {
		return fmt.Errorf("--all and a thread id are mutually exclusive")
	}

	if !force && !all {
		direction := "read"
		if markUnread {
			direction = "UNREAD"
		}
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Mark notification %s as %s?", color.YellowString("?"), threadID, direction),
			false,
		)
		if err != nil {
			return err
		}
		if !yes {
			fmt.Println("Cancelled")
			return nil
		}
	} else if !force && all {
		direction := "read"
		if markUnread {
			direction = "UNREAD"
		}
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Mark ALL notifications as %s?", color.YellowString("?"), direction),
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

	ctx, cancel := c.ctx()
	defer cancel()

	if all {
		req := types.NotificationMarkAllReadRequest{Unread: markUnread}
		if err := c.API.Put(ctx, api.NotificationsMarkAllRead(), req, nil); err != nil {
			return fmt.Errorf("marking all notifications read: %w", err)
		}
		green := color.New(color.FgGreen).SprintFunc()
		if markUnread {
			fmt.Printf("%s All notifications marked as UNREAD\n", green("✓"))
		} else {
			fmt.Printf("%s All notifications marked as read\n", green("✓"))
		}
		return nil
	}

	req := types.NotificationMarkReadRequest{Unread: markUnread}
	if err := c.API.Put(ctx, api.NotificationMarkRead(threadID), req, nil); err != nil {
		return fmt.Errorf("marking notification %s read: %w", threadID, err)
	}
	green := color.New(color.FgGreen).SprintFunc()
	action := "read"
	if markUnread {
		action = "UNREAD"
	}
	fmt.Printf("%s Notification %s marked as %s\n", green("✓"), threadID, action)
	return nil
}
