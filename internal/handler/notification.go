// Package handler — notification subpackage logic.
//
// Notifications are user-scoped (no {owner}/{repo} in the URL), so the
// surface here is smaller than milestone / label: list + count + mark
// read. Unlike label / ssh-key, the notification verbs have no need
// for repo resolution or confirmation prompts (mark-read is reversible
// by re-running with unread=true).
package handler

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
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
func NotificationList(unreadFilter string, limit int) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var threads []types.NotificationThread
	if err := client.Get(ctx, path, &threads); err != nil {
		return fmt.Errorf("listing notifications: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return PrintJSON(threads)
	}

	if len(threads) == 0 {
		if unreadFilter == "true" {
			fmt.Println("No unread notifications. You're all caught up.")
		} else {
			fmt.Println("No notifications.")
		}
		return nil
	}

	w := output.NewTableWriter()
	fmt.Fprintln(w, "ID\tREPO\tTYPE\tSUBJECT\tWHEN")
	for _, t := range threads {
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
		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\n",
			marker, t.ID, repoName, t.Subject.Type, subject, when)
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d thread(s) (* = unread)\n", len(threads))
	return nil
}

// NotificationCount runs the notification-count operation and prints
// just the unread count. In JSON mode it prints the full count object.
func NotificationCount() error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var resp types.NotificationUnreadCount
	if err := client.Get(ctx, api.NotificationsCount(), &resp); err != nil {
		return fmt.Errorf("getting notification count: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return PrintJSON(resp)
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
func NotificationRead(threadID string, all bool, markUnread bool, force bool) error {
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

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if all {
		req := types.NotificationMarkAllReadRequest{Unread: markUnread}
		if err := client.Put(ctx, api.NotificationsMarkAllRead(), req, nil); err != nil {
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
	if err := client.Put(ctx, api.NotificationMarkRead(threadID), req, nil); err != nil {
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
