package alias

import (
	"strings"
)

// ResolveArgs rewrites args so that if args[1] is a user-defined
// alias, it is replaced by the alias's expansion. The rest of the
// args (flags, positionals) are preserved in order.
//
// Example:
//
//	[ "gitee", "co", "123" ]  +  aliases={"co":"pr checkout"}
//	-> [ "gitee", "pr", "checkout", "123" ]
//
// `builtins` is the set of cobra command names that already exist
// on the root. Anything in `builtins` is left alone — built-ins
// always win, so a user who happens to define `alias set issue "x"`
// still gets the real `gitee issue` command.
//
// If args[1] is a flag (starts with '-'), the function is a no-op:
// aliases don't conflict with flags because the only root flags all
// start with '-' (--agent-help, --output, etc.).
//
// If the alias file is unreadable, ResolveArgs is a no-op (returns
// args unchanged). Failing to expand an alias is never fatal — the
// worst case is the user sees "unknown command" and can debug.
func ResolveArgs(args []string, builtins map[string]bool) []string {
	if len(args) < 2 {
		return args
	}
	head := args[1]
	// A flag like "--agent-help" or "-h" is never an alias.
	if strings.HasPrefix(head, "-") {
		return args
	}
	// Built-in commands take precedence over aliases.
	if builtins[head] {
		return args
	}
	m, err := Load()
	if err != nil {
		// Silent fallback: alias file is corrupt. The user will see
		// the normal "unknown command" message and can investigate.
		return args
	}
	expansion, ok := m[head]
	if !ok {
		return args
	}
	// Splice expansion in: keep args[0] (the program name), insert
	// the expansion tokens, then keep the rest of the user's args.
	expansionTokens := strings.Fields(expansion)
	if len(expansionTokens) == 0 {
		// Guarded against by Set/Set expansion validation, but be
		// defensive: a malformed file with an empty expansion
		// shouldn't crash the resolver.
		return args
	}
	out := make([]string, 0, 1+len(expansionTokens)+(len(args)-2))
	out = append(out, args[0])
	out = append(out, expansionTokens...)
	if len(args) > 2 {
		out = append(out, args[2:]...)
	}
	return out
}

// BuiltinNames is a small helper to build the builtins map from a
// slice of cobra-style command names. Used by main.go to wire the
// resolver without having to know about cobra types.
func BuiltinNames(names []string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}
