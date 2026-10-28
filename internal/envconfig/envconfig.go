// Package envconfig is the single source of truth for the environment
// variables gitee-cli recognizes.
//
// Why a package instead of a list inside cmd: the same set of names has to
// be visible to three consumers, and before this package existed each kept
// its own copy —
//
//   - cmd/env.go       (which flag each variable provides a default for),
//   - cmd/root.go      (the per-flag default helpers used at registration),
//   - cmd/help_topics  (the `gitee help environment` table).
//
// Four hand-maintained copies is how the docs drifted from reality in the
// first place (GITEE_INSECURE / GITEE_EDITOR / csv were documented but never
// implemented). Both cmd and help_topics now derive from Variables here, and
// envconfig_test.go freezes the canonical set so silently dropping a row
// fails CI.
package envconfig

import (
	"os"
	"strconv"
	"strings"
)

// Kind describes how an environment value is parsed.
type Kind int

const (
	// String passes the raw (trimmed) value through.
	String Kind = iota
	// Int requires an integer; a non-numeric value is a hard error when
	// applied (a typo like GITEE_TIMEOUT=30s must not silently become 30).
	Int
	// Bool requires a boolean word (1/true/yes/on, 0/false/no/off).
	Bool
	// Presence turns any non-empty value into true (NO_COLOR semantics:
	// the spec says presence, not parseability, disables color).
	Presence
)

// Variable describes one recognized environment variable.
type Variable struct {
	// Name is the environment variable name, e.g. "GITEE_TIMEOUT".
	Name string
	// Flag is the root persistent flag this variable provides a DEFAULT
	// value for, e.g. "timeout". Empty for variables read directly by a
	// subsystem (see Reader).
	Flag string
	// Kind is how the value is parsed.
	Kind Kind
	// Default is the user-facing default shown by `gitee help environment`
	// (the flag's own default, not the env var's).
	Default string
	// Description is the one-line explanation shown by
	// `gitee help environment`.
	Description string
	// Reader names the subsystem that reads the variable directly. Only
	// meaningful when Flag is empty; documents where the implementation
	// lives so the registry can be audited.
	Reader string
}

// Variables is the canonical, ordered set of environment variables
// gitee-cli reads. Order is the order `gitee help environment` renders.
//
// Locked by TestCanonicalVariableSet: adding, removing, or renaming a row
// must update that test and the user-facing docs in the same commit.
var Variables = []Variable{
	// --- auth ---
	{
		Name:        "GITEE_TOKEN",
		Kind:        String,
		Default:     "(none — required for most commands)",
		Description: "Personal access token. Equivalent to --token. Falls back to settings.json when unset.",
		Reader:      "internal/config.GetToken",
	},
	// --- host / enterprise ---
	{
		Name:        "GITEE_HOSTNAME",
		Flag:        "hostname",
		Kind:        String,
		Default:     "gitee.com",
		Description: "Gitee hostname. Override for Gitee Enterprise (e.g. git.gitee.work). Equivalent to --hostname; overrides the config file's active host.",
	},
	{
		Name:        "GITEE_API_URL",
		Flag:        "api-url",
		Kind:        String,
		Default:     "https://{hostname}/api/v5",
		Description: "Full API base URL. Use when your enterprise deployment has a non-standard API path. Equivalent to --api-url.",
	},
	// --- output ---
	{
		Name:        "GITEE_OUTPUT",
		Flag:        "output",
		Kind:        String,
		Default:     "table",
		Description: "Default output format: table|json. Equivalent to --output/-o. Any other value is rejected. (csv was never implemented and is not accepted.)",
	},
	{
		Name:        "NO_COLOR",
		Flag:        "no-color",
		Kind:        Presence,
		Default:     "(unset — colors enabled when stdout is a TTY)",
		Description: "Any non-empty value disables ANSI color codes in output. Equivalent to --no-color. Standard no-color.org convention.",
	},
	// --- pagination / limits ---
	{
		Name:        "GITEE_PAGINATE",
		Flag:        "paginate",
		Kind:        Bool,
		Default:     "(unset — single page only)",
		Description: "Set to 1/true/yes to enable --paginate behaviour by default (walks all pages). Equivalent to --paginate/-P.",
	},
	{
		Name:        "GITEE_LIMIT",
		Flag:        "limit",
		Kind:        Int,
		Default:     "30",
		Description: "Default --limit value (max results). Equivalent to --limit; applies to list commands that define their own --limit too.",
	},
	// --- debug / network ---
	{
		Name:        "GITEE_VERBOSE",
		Flag:        "verbose",
		Kind:        Bool,
		Default:     "(unset — quiet)",
		Description: "Set to 1/true/yes to print HTTP request/response details on stderr. Equivalent to --verbose/-v.",
	},
	{
		Name:        "GITEE_TIMEOUT",
		Flag:        "timeout",
		Kind:        Int,
		Default:     "30",
		Description: "HTTP timeout in seconds. Equivalent to --timeout. Non-integer values are a hard error.",
	},
	// --- extensions ---
	{
		Name:        "GITEE_NO_EXTENSIONS",
		Flag:        "no-extensions",
		Kind:        Bool,
		Default:     "(unset — extensions loaded)",
		Description: "Set to 1/true/yes to skip the dynamic extension loader on startup. Equivalent to --no-extensions.",
	},
	{
		Name:        "GITEE_EXT_DIR",
		Kind:        String,
		Default:     "~/.gitee-cli/extensions",
		Description: "Override the extension install directory. Mainly useful for portable dotfiles / test isolation.",
		Reader:      "internal/extension.ExtDir",
	},
	// --- aliases ---
	{
		Name:        "GITEE_ALIASES_FILE",
		Kind:        String,
		Default:     "~/.gitee-cli/aliases.json",
		Description: "Override the aliases JSON file location. Useful for portable dotfiles and CI.",
		Reader:      "internal/alias",
	},
	// --- browser ---
	{
		Name:        "BROWSER",
		Kind:        String,
		Default:     "platform default",
		Description: "Browser binary used by commands that open a browser (e.g. `gitee repo browse`). Standard cross-tool variable.",
		Reader:      "cmd/repo/browse.go",
	},
}

// FlagVariables returns the subset of Variables that provide defaults for a
// flag (Flag != ""), in the same order as Variables.
func FlagVariables() []Variable {
	out := make([]Variable, 0, len(Variables))
	for _, v := range Variables {
		if v.Flag != "" {
			out = append(out, v)
		}
	}
	return out
}

// ParseBool accepts the boolean spellings users expect in env vars:
// 1/0, true/false, yes/no, on/off (case-insensitive).
func ParseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "t", "true", "y", "yes", "on":
		return true, nil
	case "0", "f", "false", "n", "no", "off":
		return false, nil
	}
	return false, errNotBool
}

type envBoolError struct{}

func (envBoolError) Error() string {
	return "expected a boolean (1/0, true/false, yes/no, on/off)"
}

// errNotBool is the shared parse error (value-less on purpose: callers wrap
// it with the variable name).
var errNotBool = envBoolError{}

// Lookup returns the trimmed value of the variable and whether it is set to
// a non-empty value.
func Lookup(name string) (string, bool) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return "", false
	}
	raw = strings.TrimSpace(raw)
	return raw, raw != ""
}

// StringDefault returns the variable's value when set and non-empty,
// otherwise def. These *Default helpers apply the environment layer at flag
// REGISTRATION time so env-provided defaults are visible in --help.
func StringDefault(name, def string) string {
	if v, ok := Lookup(name); ok {
		return v
	}
	return def
}

// IntDefault returns the variable parsed as an integer, or def when it is
// unset or unparseable. Invalid values are surfaced as errors by the
// runtime resolution pass (cmd.applyEnvDefaults), not silently ignored.
func IntDefault(name string, def int) int {
	raw, ok := Lookup(name)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

// BoolDefault returns the variable parsed as a boolean, or def when it is
// unset or unparseable (the runtime pass reports the error).
func BoolDefault(name string, def bool) bool {
	raw, ok := Lookup(name)
	if !ok {
		return def
	}
	b, err := ParseBool(raw)
	if err != nil {
		return def
	}
	return b
}

// PresenceDefault reports whether the variable is set to any non-empty value
// (NO_COLOR semantics).
func PresenceDefault(name string) bool {
	_, ok := Lookup(name)
	return ok
}
