package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// GroupRunE is the RunE installed on every command-group umbrella — verbs
// that exist only to group subcommands (`gitee repo`, `gitee pr`,
// `gitee issue label`, `gitee repo star`, …).
//
// Behavior:
//
//   - No arguments: print the group's help and exit 0. This keeps the
//     discovery path (`gitee repo` shows what repo can do).
//   - With arguments: return an error instead of silently printing help
//     and exiting 0.
//
// The second case used to be a silent trap: cobra prints help and exits 0
// for a non-runnable parent command, so `gitee repo push a.txt b.txt`,
// `gitee pr creat 1` or `gitee issue labl list` looked like successes
// while doing nothing. The error now names the best correction:
//
//  1. the group's own name is also a verb under it (`repo protect master`
//     → `repo protect protect master`);
//  2. a same-named verb lives deeper in the tree (`repo push x` →
//     `repo file push x`);
//  3. a sibling verb is within a small edit distance (`pr creat` →
//     `pr create`, `issue labl` → `issue label`);
//  4. otherwise it lists the available subcommands.
//
// It never prints the group's help on the error path: a stale script must
// fail loudly, not pretend it succeeded.
func GroupRunE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return c.Help()
	}

	names := visibleChildNames(c)
	remaining := strings.Join(args[1:], " ")
	suggestion := ""

	// 0. The group's own name is also a verb under it: `repo star <x>`,
	// `repo watch <x>`, `repo protect <branch>` are the old flat syntax,
	// and the correction is always `<group> <group> <x>` regardless of
	// what the argument looks like (it is usually a repo/branch, not a
	// subcommand name).
	if child := findChild(c, c.Name()); child != nil {
		suggestion = joinCommand(child.CommandPath(), strings.Join(args, " "))
	} else if child := findChild(c, args[0]); child != nil {
		// 1. Exact subcommand name used as an argument.
		suggestion = joinCommand(child.CommandPath(), remaining)
	} else if target := findDescendant(c, args[0]); target != nil {
		// 2. Same-named verb somewhere deeper (e.g. repo push lives at
		// repo file push). unstar/unwatch are direct children, so cobra
		// dispatches those before GroupRunE.
		suggestion = joinCommand(target.CommandPath(), remaining)
	} else if target := closestChild(c, args[0]); target != "" {
		// 3. Typo of a direct child (Levenshtein distance <= 2).
		suggestion = joinCommand(c.CommandPath()+" "+target, remaining)
	}

	var hint string
	switch {
	case suggestion != "":
		hint = fmt.Sprintf("did you mean `%s`?", suggestion)
	case len(names) > 0:
		hint = fmt.Sprintf("run `%s <subcommand>` (available: %s)", c.CommandPath(), strings.Join(names, ", "))
	default:
		hint = fmt.Sprintf("run `%s --help`", c.CommandPath())
	}

	return fmt.Errorf("%q is a command group and takes no direct arguments; "+
		"%q was not interpreted as a subcommand. %s",
		c.CommandPath(), args[0], hint)
}

// joinCommand appends the given argument string to a command path, handling
// the empty case without a trailing space.
func joinCommand(path, args string) string {
	if args == "" {
		return path
	}
	return path + " " + args
}

// visibleChildNames returns the sorted names of c's non-hidden direct
// subcommands.
func visibleChildNames(c *cobra.Command) []string {
	children := c.Commands()
	names := make([]string, 0, len(children))
	for _, sub := range children {
		if !sub.Hidden {
			names = append(names, sub.Name())
		}
	}
	sort.Strings(names)
	return names
}

// findChild returns c's non-hidden direct child named name, or nil.
func findChild(c *cobra.Command, name string) *cobra.Command {
	for _, sub := range c.Commands() {
		if !sub.Hidden && sub.Name() == name {
			return sub
		}
	}
	return nil
}

// findDescendant does a depth-first search below c (excluding c itself) for
// a non-hidden command named name. This is what resolves reparented verbs
// that moved into a sub-group, e.g. `gitee repo push` lives at
// `gitee repo file push`, and the old `gitee org member` now lives at
// `gitee org members`. `gitee repo unstar` and `gitee repo unwatch` are
// direct aliases and are not handled here.
func findDescendant(c *cobra.Command, name string) *cobra.Command {
	for _, sub := range c.Commands() {
		if sub.Hidden {
			continue
		}
		if sub.Name() == name {
			return sub
		}
		if len(sub.Commands()) > 0 {
			if found := findDescendant(sub, name); found != nil {
				return found
			}
		}
	}
	return nil
}

// closestChild returns the name of the direct child closest to the user's
// mistyped word, when it is within edit distance 2 (one typo, one
// transposition, or one missing/extra letter). Empty when nothing is close.
func closestChild(c *cobra.Command, typo string) string {
	best := ""
	bestDist := 3 // only suggestions with distance <= 2 are helpful
	for _, sub := range c.Commands() {
		if sub.Hidden {
			continue
		}
		if d := editDistance(strings.ToLower(typo), strings.ToLower(sub.Name())); d < bestDist {
			bestDist = d
			best = sub.Name()
		}
	}
	return best
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}
	prev := make([]int, len(br)+1)
	cur := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		cur[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(br)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
