package envconfig

import (
	"os"
	"testing"
)

// canonicalVariables is the frozen public contract: the exact set of
// environment variables `gitee help environment` documents and the runtime
// env layer implements, in the documented order.
//
// This list is intentionally a second copy of Variables. It is the lock that
// makes deleting (or silently renaming) a registry row a CI failure instead
// of a quiet documentation change. Update it — and the docs — deliberately.
var canonicalVariables = []string{
	"GITEE_TOKEN",
	"GITEE_HOSTNAME",
	"GITEE_API_URL",
	"GITEE_OUTPUT",
	"NO_COLOR",
	"GITEE_PAGINATE",
	"GITEE_LIMIT",
	"GITEE_VERBOSE",
	"GITEE_TIMEOUT",
	"GITEE_NO_EXTENSIONS",
	"GITEE_EXT_DIR",
	"GITEE_ALIASES_FILE",
	"BROWSER",
}

// TestCanonicalVariableSet freezes the registry. Adding/removing/renaming a
// variable without updating canonicalVariables (and therefore the maintained
// contract) fails here.
func TestCanonicalVariableSet(t *testing.T) {
	got := make([]string, 0, len(Variables))
	for _, v := range Variables {
		got = append(got, v.Name)
	}

	if len(got) != len(canonicalVariables) {
		t.Fatalf("registry has %d variables, canonical set has %d\ngot: %v\nwant: %v",
			len(got), len(canonicalVariables), got, canonicalVariables)
	}
	for i := range canonicalVariables {
		if got[i] != canonicalVariables[i] {
			t.Errorf("registry[%d] = %q, want %q (keep Variables and canonicalVariables in sync, including order)",
				i, got[i], canonicalVariables[i])
		}
	}
}

// TestEveryVariableIsDocumented enforces that a registry row is always
// user-facing documentation, and that direct-read variables point at their
// implementation so the registry can be audited.
func TestEveryVariableIsDocumented(t *testing.T) {
	for _, v := range Variables {
		if v.Name == "" {
			t.Error("variable with empty Name")
		}
		if v.Default == "" {
			t.Errorf("%s: empty Default (help environment needs a default column)", v.Name)
		}
		if v.Description == "" {
			t.Errorf("%s: empty Description", v.Name)
		}
		if v.Flag == "" && v.Reader == "" {
			t.Errorf("%s: Flag-less (direct-read) variable must document Reader", v.Name)
		}
		if v.Flag != "" && v.Reader != "" {
			t.Errorf("%s: has both Flag and Reader — pick one source", v.Name)
		}
	}
}

// TestFlagVariablesShape pins the env→flag projection consumed by
// cmd.applyEnvDefaults, including each variable's parse kind.
func TestFlagVariablesShape(t *testing.T) {
	want := map[string]Kind{
		"GITEE_HOSTNAME":      String,
		"GITEE_API_URL":       String,
		"GITEE_OUTPUT":        String,
		"GITEE_VERBOSE":       Bool,
		"GITEE_TIMEOUT":       Int,
		"GITEE_LIMIT":         Int,
		"GITEE_PAGINATE":      Bool,
		"GITEE_NO_EXTENSIONS": Bool,
		"NO_COLOR":            Presence,
	}
	got := make(map[string]Kind, len(want))
	for _, v := range FlagVariables() {
		got[v.Name] = v.Kind
	}
	if len(got) != len(want) {
		t.Errorf("FlagVariables() has %d entries, want %d: %v", len(got), len(want), got)
	}
	for name, kind := range want {
		if got[name] != kind {
			t.Errorf("%s kind = %v, want %v", name, got[name], kind)
		}
	}
}

func TestParseBool(t *testing.T) {
	for _, raw := range []string{"1", "t", "true", "TRUE", "y", "yes", "on", " Yes "} {
		if b, err := ParseBool(raw); err != nil || !b {
			t.Errorf("ParseBool(%q) = %v, %v; want true, nil", raw, b, err)
		}
	}
	for _, raw := range []string{"0", "f", "false", "n", "no", "off"} {
		if b, err := ParseBool(raw); err != nil || b {
			t.Errorf("ParseBool(%q) = %v, %v; want false, nil", raw, b, err)
		}
	}
	for _, raw := range []string{"maybe", "2", "truthy"} {
		if _, err := ParseBool(raw); err == nil {
			t.Errorf("ParseBool(%q) should error", raw)
		}
	}
}

func TestDefaultsHelpers(t *testing.T) {
	t.Setenv("ENVCONFIG_TEST_STRING", " hello ")
	t.Setenv("ENVCONFIG_TEST_INT", "7")
	t.Setenv("ENVCONFIG_TEST_BOOL", "yes")
	t.Setenv("ENVCONFIG_TEST_PRESENCE", "0") // presence, not truthiness
	t.Setenv("ENVCONFIG_TEST_EMPTY", "")

	if got := StringDefault("ENVCONFIG_TEST_STRING", "def"); got != "hello" {
		t.Errorf("StringDefault = %q, want trimmed value", got)
	}
	if got := StringDefault("ENVCONFIG_TEST_MISSING", "def"); got != "def" {
		t.Errorf("StringDefault(missing) = %q, want default", got)
	}
	if got := StringDefault("ENVCONFIG_TEST_EMPTY", "def"); got != "def" {
		t.Errorf("StringDefault(empty) = %q, want default (empty means unset)", got)
	}
	if got := IntDefault("ENVCONFIG_TEST_INT", 30); got != 7 {
		t.Errorf("IntDefault = %d, want 7", got)
	}
	if got := IntDefault("ENVCONFIG_TEST_STRING", 30); got != 30 {
		t.Errorf("IntDefault(unparseable) = %d, want fallback 30", got)
	}
	if !BoolDefault("ENVCONFIG_TEST_BOOL", false) {
		t.Error("BoolDefault should parse yes as true")
	}
	if !PresenceDefault("ENVCONFIG_TEST_PRESENCE") {
		t.Error("PresenceDefault should be true for any non-empty value, including 0")
	}
	if PresenceDefault("ENVCONFIG_TEST_EMPTY") {
		t.Error("PresenceDefault should be false for an empty value")
	}
}

// TestLookupEmptyMeansUnset documents the one subtle semantic shared by the
// helpers: FOO= (set but empty) is treated like unset.
func TestLookupEmptyMeansUnset(t *testing.T) {
	os.Setenv("ENVCONFIG_TEST_EMPTY", "")
	defer os.Unsetenv("ENVCONFIG_TEST_EMPTY")
	if _, ok := Lookup("ENVCONFIG_TEST_EMPTY"); ok {
		t.Error("Lookup should report empty-but-set as not present")
	}
}
