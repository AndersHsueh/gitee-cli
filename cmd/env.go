package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"gitee-cli/internal/envconfig"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// This file implements the environment-variable layer of the
// flag → env → config → default priority chain documented in
// `gitee help environment`.
//
// The canonical list of variables does NOT live here — it lives in
// internal/envconfig, together with the descriptions rendered by the help
// topic. This file only consumes that registry: envFlagDefaults derives the
// env→flag mapping from it, and applyEnvDefaults applies it once per
// invocation from root's PersistentPreRunE.
//
// An env var only provides a DEFAULT for its flag: if the user passed the
// flag explicitly, the flag wins (pflag's Changed bit is the discriminator,
// exactly like ResolveHostname does for --hostname).

// envFlagDefault is the env→flag projection of envconfig.Variable used by
// applyEnvDefaults at run time.
type envFlagDefault struct {
	Env  string
	Flag string
	Kind envconfig.Kind
}

// envFlagDefaults is derived from envconfig.FlagVariables() so the set of
// env-overridable flags can never drift from the documented variable list.
var envFlagDefaults = buildEnvFlagDefaults()

// buildEnvFlagDefaults projects the canonical env registry onto the flags
// whose defaults it overrides.
func buildEnvFlagDefaults() []envFlagDefault {
	vars := envconfig.FlagVariables()
	out := make([]envFlagDefault, 0, len(vars))
	for _, v := range vars {
		out = append(out, envFlagDefault{Env: v.Name, Flag: v.Flag, Kind: v.Kind})
	}
	return out
}

// applyEnvDefaults overlays environment variables onto flags the user did
// not set explicitly on the command line.
//
// It is called from root's PersistentPreRunE, which runs before every
// command's RunE and before the per-command PreRunE hooks that consume
// RuntimeFromCmd. Invalid numeric/boolean values are returned as errors
// (non-zero exit) instead of being silently ignored: an env var the user
// deliberately set should never be a no-op.
//
// The executed command is used to resolve each flag so that a command
// with its own local flag (e.g. `repo list --limit`) is overridden too,
// not just the root persistent flag of the same name.
func applyEnvDefaults(c *cobra.Command) error {
	for _, m := range envFlagDefaults {
		// Note: Lookup does NOT treat an empty-but-set variable (FOO=) as
		// present; empty is indistinguishable from unset for defaults.
		raw, ok := envLookup(m.Env)
		if !ok {
			continue
		}

		f := envTargetFlag(c, m.Flag)
		if f == nil || f.Changed {
			continue
		}

		var value string
		switch m.Kind {
		case envconfig.String:
			value = raw
		case envconfig.Int:
			if _, err := strconv.Atoi(raw); err != nil {
				return fmt.Errorf("invalid %s=%q: expected an integer", m.Env, raw)
			}
			value = raw
		case envconfig.Bool:
			b, err := envconfig.ParseBool(raw)
			if err != nil {
				return fmt.Errorf("invalid %s=%q: %w", m.Env, raw, err)
			}
			value = strconv.FormatBool(b)
		case envconfig.Presence:
			value = "true"
		}

		if err := f.Value.Set(value); err != nil {
			return fmt.Errorf("invalid %s=%q: %w", m.Env, raw, err)
		}

		// GITEE_LIMIT also seeds the root --limit global, because list
		// commands split into two families: those with their own local
		// --limit flag (repo list, label list, …) read the local flag,
		// while those without one (pr list, issue list, status) read
		// GlobalLimit through cmdkit.Runtime. Setting both keeps
		// GITEE_LIMIT effective for every list command.
		if m.Flag == "limit" {
			if pf := rootPersistentFlag(c, "limit"); pf != nil && !pf.Changed {
				if err := pf.Value.Set(value); err != nil {
					return fmt.Errorf("invalid %s=%q: %w", m.Env, raw, err)
				}
			}
		}
	}
	return nil
}

// envLookup reads an env var, returning the trimmed value and whether it is
// set to a non-empty value. Delegates to envconfig so help/docs and runtime
// agree on what "set" means.
func envLookup(name string) (string, bool) {
	return envconfig.Lookup(name)
}

// envTargetFlag resolves the flag an env var should seed: the executed
// command's own/local flag when present (cobra merges inherited persistent
// flags into c.Flags()), else the root persistent flag.
//
// We deliberately go through the command tree (c.Root()) instead of the
// package-level rootCmd variable: referencing rootCmd from a function used
// in rootCmd's own initializer (the PersistentPreRunE closure) would be an
// initialization cycle.
func envTargetFlag(c *cobra.Command, name string) *pflag.Flag {
	if c != nil {
		if f := c.Flags().Lookup(name); f != nil {
			return f
		}
	}
	return rootPersistentFlag(c, name)
}

// rootPersistentFlag returns the root command's persistent flag named name,
// if any.
func rootPersistentFlag(c *cobra.Command, name string) *pflag.Flag {
	if c == nil {
		return nil
	}
	if root := c.Root(); root != nil {
		return root.PersistentFlags().Lookup(name)
	}
	return nil
}

// parseEnvBool accepts the boolean spellings users expect in env vars.
// Kept as a thin wrapper because the strict parser is part of the
// flag-resolution contract exercised by this package's tests; the
// implementation (and its error text) lives in envconfig.
func parseEnvBool(raw string) (bool, error) {
	return envconfig.ParseBool(raw)
}

// validOutputFormats are the formats gitee-cli actually implements.
// csv/yaml were documented for a long time but never implemented; they
// are rejected loudly instead of silently falling back to table.
var validOutputFormats = []string{"table", "json"}

// validateOutputFormat normalizes and validates the --output/-o value.
// It accepts case-insensitive input and returns the canonical lowercase
// form. Anything other than table/json is an error: silently printing a
// table when the caller asked for csv breaks agents and scripts, which
// cannot detect the substitution.
func validateOutputFormat(v string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(v))
	for _, valid := range validOutputFormats {
		if normalized == valid {
			return normalized, nil
		}
	}
	return "", fmt.Errorf("invalid output format %q: valid values are table and json", v)
}
