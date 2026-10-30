package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

// Tests for the flag → env → config → default resolution layer:
//
//   - unit tests exercise applyEnvDefaults / validateOutputFormat /
//     parseEnvBool / ResolveHostname directly;
//   - integration tests shell out to a freshly-built binary so the real
//     Cobra PreRunE pipeline (where the checks are wired) is covered,
//     including the non-zero exit codes users and agents depend on.

// snapshotAndRestoreGlobals saves the package-level flag values mutated by
// applyEnvDefaults and restores them when the test finishes, so one test's
// env cannot leak into another's assertions.
func snapshotAndRestoreGlobals(t *testing.T) {
	t.Helper()
	type snapshot struct {
		hostname, apiURL, output                 string
		verbose, noColor, paginate, noExtensions bool
		timeout, limit                           int
	}
	saved := snapshot{
		hostname:     GlobalHostname,
		apiURL:       GlobalAPIURL,
		output:       GlobalOutput,
		verbose:      GlobalVerbose,
		noColor:      NoColor,
		paginate:     GlobalPaginate,
		noExtensions: GlobalNoExtensions,
		timeout:      GlobalTimeout,
		limit:        GlobalLimit,
	}
	t.Cleanup(func() {
		GlobalHostname = saved.hostname
		GlobalAPIURL = saved.apiURL
		GlobalOutput = saved.output
		GlobalVerbose = saved.verbose
		NoColor = saved.noColor
		GlobalPaginate = saved.paginate
		GlobalNoExtensions = saved.noExtensions
		GlobalTimeout = saved.timeout
		GlobalLimit = saved.limit
	})
}

func TestParseEnvBool(t *testing.T) {
	trueWords := []string{"1", "t", "true", "TRUE", "y", "yes", "on", " Yes "}
	for _, v := range trueWords {
		got, err := parseEnvBool(v)
		if err != nil || !got {
			t.Errorf("parseEnvBool(%q) = %v, %v; want true, nil", v, got, err)
		}
	}
	falseWords := []string{"0", "f", "false", "FALSE", "n", "no", "off"}
	for _, v := range falseWords {
		got, err := parseEnvBool(v)
		if err != nil || got {
			t.Errorf("parseEnvBool(%q) = %v, %v; want false, nil", v, got, err)
		}
	}
	for _, v := range []string{"maybe", "2", "-1", "truthy"} {
		if _, err := parseEnvBool(v); err == nil {
			t.Errorf("parseEnvBool(%q) should return an error", v)
		}
	}
}

func TestValidateOutputFormat(t *testing.T) {
	for input, want := range map[string]string{
		"table":  "table",
		"json":   "json",
		"JSON":   "json",
		" Json ": "json",
	} {
		got, err := validateOutputFormat(input)
		if err != nil || got != want {
			t.Errorf("validateOutputFormat(%q) = %q, %v; want %q, nil", input, got, err, want)
		}
	}
	for _, input := range []string{"", "csv", "CSV", "yaml", "xml", "ndjson"} {
		if _, err := validateOutputFormat(input); err == nil {
			t.Errorf("validateOutputFormat(%q) should be rejected (csv/yaml were never implemented)", input)
		}
	}
}

// TestApplyEnvDefaultsSetsGlobals is the unit-level proof that each of the
// nine env-overridable variables actually reaches the flag/global it claims
// to override in `gitee help environment`.
func TestApplyEnvDefaultsSetsGlobals(t *testing.T) {
	snapshotAndRestoreGlobals(t)

	for k, v := range map[string]string{
		"GITEE_HOSTNAME":      "enterprise.example.com",
		"GITEE_API_URL":       "https://enterprise.example.com/api/v5",
		"GITEE_OUTPUT":        "json",
		"GITEE_VERBOSE":       "yes",
		"GITEE_TIMEOUT":       "7",
		"GITEE_LIMIT":         "5",
		"GITEE_PAGINATE":      "on",
		"GITEE_NO_EXTENSIONS": "1",
		"NO_COLOR":            "1",
	} {
		t.Setenv(k, v)
	}

	if err := applyEnvDefaults(rootCmd); err != nil {
		t.Fatalf("applyEnvDefaults: %v", err)
	}

	if GlobalHostname != "enterprise.example.com" {
		t.Errorf("GlobalHostname = %q, want env value", GlobalHostname)
	}
	if GlobalAPIURL != "https://enterprise.example.com/api/v5" {
		t.Errorf("GlobalAPIURL = %q, want env value", GlobalAPIURL)
	}
	if GlobalOutput != "json" {
		t.Errorf("GlobalOutput = %q, want env value", GlobalOutput)
	}
	if !GlobalVerbose {
		t.Error("GlobalVerbose = false, want true from GITEE_VERBOSE=yes")
	}
	if GlobalTimeout != 7 {
		t.Errorf("GlobalTimeout = %d, want 7", GlobalTimeout)
	}
	if GlobalLimit != 5 {
		t.Errorf("GlobalLimit = %d, want 5", GlobalLimit)
	}
	if !GlobalPaginate {
		t.Error("GlobalPaginate = false, want true from GITEE_PAGINATE=on")
	}
	if !GlobalNoExtensions {
		t.Error("GlobalNoExtensions = false, want true from GITEE_NO_EXTENSIONS=1")
	}
	if !NoColor {
		t.Error("NoColor = false, want true from NO_COLOR=1")
	}
}

// TestApplyEnvDefaultsRejectsBadValues guards the "no silent fallback" rule:
// a value the user deliberately put in the environment must either work or
// fail loudly, never be ignored.
func TestApplyEnvDefaultsRejectsBadValues(t *testing.T) {
	cases := []struct{ env, val, wantSubstr string }{
		{"GITEE_TIMEOUT", "soon", "GITEE_TIMEOUT"},
		{"GITEE_LIMIT", "ten", "GITEE_LIMIT"},
		{"GITEE_VERBOSE", "maybe", "GITEE_VERBOSE"},
		{"GITEE_PAGINATE", "sometimes", "GITEE_PAGINATE"},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			snapshotAndRestoreGlobals(t)
			t.Setenv(tc.env, tc.val)
			err := applyEnvDefaults(rootCmd)
			if err == nil {
				t.Fatalf("%s=%q should be a hard error", tc.env, tc.val)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("error %q should name %s", err, tc.wantSubstr)
			}
		})
	}
}

// TestApplyEnvDefaultsSkipsExplicitFlags proves the priority rule: an
// explicitly-passed flag wins over the environment. It uses an isolated
// cobra tree so it cannot leave a Changed bit on the real rootCmd.
func TestApplyEnvDefaultsSkipsExplicitFlags(t *testing.T) {
	root := &cobra.Command{Use: "gitee"}
	var output string
	root.PersistentFlags().StringVar(&output, "output", "table", "")
	child := &cobra.Command{Use: "list"}
	root.AddCommand(child)

	// Simulate `gitee list --output json`: FlagSet.Set marks the flag Changed.
	if err := root.PersistentFlags().Set("output", "json"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	t.Setenv("GITEE_OUTPUT", "table")

	if err := applyEnvDefaults(child); err != nil {
		t.Fatalf("applyEnvDefaults: %v", err)
	}
	if output != "json" {
		t.Errorf("explicit --output json was overridden by GITEE_OUTPUT: got %q", output)
	}
}

// TestResolveHostnameUsesEnv verifies the documented host priority:
// --hostname > GITEE_HOSTNAME > config active host > gitee.com.
func TestResolveHostnameUsesEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no settings.json → no config active host
	t.Setenv("GITEE_HOSTNAME", "enterprise.example.com")
	if got := ResolveHostname(); got != "enterprise.example.com" {
		t.Errorf("ResolveHostname() = %q, want GITEE_HOSTNAME value", got)
	}
}

// ----- integration tests (real binary, real PreRunE pipeline) -----

// runThrowawayBinary runs the freshly-built gitee binary with an isolated
// HOME plus the given extra environment entries.
func runThrowawayBinary(t *testing.T, bin string, extraEnv []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = append([]string{
		"HOME=" + t.TempDir(),
		"PATH=/usr/bin:/bin",
	}, extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestOutputFlagRejectsUnsupportedFormats is the CLI-level contract for -o:
// only table and json are accepted; csv/yaml must exit non-zero instead of
// silently printing a table (the pre-existing bug).
func TestOutputFlagRejectsUnsupportedFormats(t *testing.T) {
	bin := buildThrowawayBinary(t)

	for _, format := range []string{"csv", "yaml", "xml"} {
		out, err := runThrowawayBinary(t, bin, nil, "version", "-o", format)
		if err == nil {
			t.Errorf("gitee version -o %s should exit non-zero; got success:\n%s", format, out)
			continue
		}
		if !strings.Contains(out, "invalid output format") {
			t.Errorf("gitee version -o %s error should explain valid formats; got:\n%s", format, out)
		}
	}

	// Valid formats must still work.
	for _, format := range []string{"table", "json", "JSON"} {
		out, err := runThrowawayBinary(t, bin, nil, "version", "-o", format)
		if err != nil {
			t.Errorf("gitee version -o %s should succeed; got %v:\n%s", format, err, out)
		}
	}
}

// TestEnvOutputIsHonored proves GITEE_OUTPUT reaches the resolved runtime:
// with GITEE_OUTPUT=json, `--agent-help` (which switches to JSON when the
// output format is json) must emit JSON instead of markdown.
func TestEnvOutputIsHonored(t *testing.T) {
	bin := buildThrowawayBinary(t)

	out, err := runThrowawayBinary(t, bin, []string{"GITEE_OUTPUT=json"}, "alias", "list", "--agent-help")
	if err != nil {
		t.Fatalf("alias list --agent-help with GITEE_OUTPUT=json failed: %v\n%s", err, out)
	}
	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "{") {
		t.Errorf("GITEE_OUTPUT=json was ignored: --agent-help did not emit JSON:\n%s", out)
	}

	// And an invalid GITEE_OUTPUT must fail loudly.
	out, err = runThrowawayBinary(t, bin, []string{"GITEE_OUTPUT=csv"}, "version")
	if err == nil {
		t.Errorf("GITEE_OUTPUT=csv should exit non-zero; got success:\n%s", out)
	}

	// An explicitly-passed flag wins over the env var.
	out, err = runThrowawayBinary(t, bin, []string{"GITEE_OUTPUT=csv"}, "version", "-o", "table")
	if err != nil {
		t.Errorf("-o table should override GITEE_OUTPUT=csv; got %v:\n%s", err, out)
	}
}

// TestEnvTimeoutRejectsNonInteger covers the "invalid env is an error, not a
// fallback" behavior end to end.
func TestEnvTimeoutRejectsNonInteger(t *testing.T) {
	bin := buildThrowawayBinary(t)
	out, err := runThrowawayBinary(t, bin, []string{"GITEE_TIMEOUT=30s"}, "version")
	if err == nil {
		t.Errorf("GITEE_TIMEOUT=30s should exit non-zero; got success:\n%s", out)
	}
	if !strings.Contains(out, "GITEE_TIMEOUT") {
		t.Errorf("error should name GITEE_TIMEOUT; got:\n%s", out)
	}
}

// TestEnvHostnameAndNoColorAreAccepted is a smoke test that documented env
// vars do not break a command that touches no network. It also guards
// against GITEE_HOSTNAME accidentally being treated as an unknown variable.
func TestEnvHostnameAndNoColorAreAccepted(t *testing.T) {
	bin := buildThrowawayBinary(t)
	out, err := runThrowawayBinary(t, bin, []string{
		"GITEE_HOSTNAME=enterprise.example.com",
		"NO_COLOR=1",
		"GITEE_PAGINATE=1",
	}, "version")
	if err != nil {
		t.Fatalf("version with documented env vars failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "gitee version") {
		t.Errorf("unexpected version output:\n%s", out)
	}
}

// ----- end-to-end env behavior against a stub Gitee API -----

// newStubAPIServer serves a Gitee-like /user/repos endpoint and records the
// per_page query params it receives, so tests can assert what the binary
// actually asked the API for.
func newStubAPIServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var perPages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/user" {
			_, _ = w.Write([]byte(`{"login":"stub-user","name":"Stub","id":1}`))
			return
		}
		mu.Lock()
		perPages = append(perPages, r.URL.Query().Get("per_page"))
		mu.Unlock()
		items := make([]map[string]any, 0, 30)
		for i := 0; i < 10; i++ {
			items = append(items, map[string]any{
				"full_name":        "owner/repo",
				"language":         "Go",
				"stargazers_count": i,
				"forks_count":      i,
				"description":      "stub",
				"updated_at":       "2026-01-01T00:00:00Z",
			})
		}
		_ = json.NewEncoder(w).Encode(items)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), perPages...)
	}
}

// TestEnvLimitCapsListResults is the end-to-end counterpart of
// TestApplyEnvDefaultsSetsGlobals for GITEE_LIMIT, covering every list
// family: cmdkit commands (repo/issue/pr list), the hand-rolled
// FetchListOrAll caller (repo commits list), and the shared cap in
// api.FetchListOrAll that makes them agree.
func TestEnvLimitCapsListResults(t *testing.T) {
	bin := buildThrowawayBinary(t)

	cases := []struct {
		name string
		args []string
	}{
		{"repo list (cmdkit)", []string{"repo", "list", "-o", "json"}},
		{"repo commits list (direct FetchListOrAll)", []string{"repo", "commits", "list", "-R", "owner/repo", "-o", "json"}},
		{"issue list (cmdkit, global -R)", []string{"issue", "list", "-R", "owner/repo", "-o", "json"}},
		{"pr list (cmdkit)", []string{"pr", "list", "-R", "owner/repo", "-o", "json"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, perPages := newStubAPIServer(t)
			out, err := runThrowawayBinary(t, bin, []string{
				"GITEE_TOKEN=stub",
				"GITEE_API_URL=" + srv.URL,
				"GITEE_LIMIT=1",
			}, tc.args...)
			if err != nil {
				t.Fatalf("%v with GITEE_LIMIT=1 failed: %v\n%s", tc.args, err, out)
			}
			var items []map[string]any
			if err := json.Unmarshal([]byte(out), &items); err != nil {
				t.Fatalf("output is not a JSON array: %v\n%s", err, out)
			}
			if len(items) != 1 {
				t.Errorf("GITEE_LIMIT=1 returned %d items, want 1 (env limit ignored)", len(items))
			}
			for _, pp := range perPages() {
				if pp != "1" {
					t.Errorf("API request used per_page=%q, want 1 (GITEE_LIMIT must bound the request)", pp)
				}
			}
		})
	}
}

// TestEnvLimitCapsStatus covers the remaining shared path: `gitee status`
// calls api.FetchListOrAll directly (not through cmdkit), so it must honor
// GITEE_LIMIT too — this is the case that exposed the duplicated cap.
// -R is passed explicitly so the test does not depend on the checkout's
// git origin being a Gitee URL (GitHub checkouts exit 0 before any API call).
func TestEnvLimitCapsStatus(t *testing.T) {
	bin := buildThrowawayBinary(t)
	srv, perPages := newStubAPIServer(t)

	out, err := runThrowawayBinary(t, bin, []string{
		"GITEE_TOKEN=stub",
		"GITEE_API_URL=" + srv.URL,
		"GITEE_LIMIT=1",
	}, "status", "-R", "owner/repo", "-o", "json")
	if err != nil {
		t.Fatalf("status with GITEE_LIMIT=1 failed: %v\n%s", err, out)
	}
	pages := perPages()
	if len(pages) == 0 {
		t.Fatal("status made no paginated API requests — stub wiring is wrong")
	}
	for _, pp := range pages {
		if pp != "1" {
			t.Errorf("status request used per_page=%q, want 1 (GITEE_LIMIT ignored on a direct FetchListOrAll path)", pp)
		}
	}
}

// TestEnvVerboseTracesToStderr locks two things at once:
//   - GITEE_VERBOSE=1 enables HTTP tracing (no flag passed), and
//   - the trace goes to stderr so stdout stays valid JSON.
func TestEnvVerboseTracesToStderr(t *testing.T) {
	bin := buildThrowawayBinary(t)
	srv, _ := newStubAPIServer(t)

	out, err := runThrowawayBinary(t, bin, []string{
		"GITEE_TOKEN=stub",
		"GITEE_API_URL=" + srv.URL,
		"GITEE_VERBOSE=1",
	}, "repo", "list", "-o", "json")
	if err != nil {
		t.Fatalf("repo list with GITEE_VERBOSE=1 failed: %v", err)
	}
	if !strings.Contains(out, "> GET") {
		t.Errorf("GITEE_VERBOSE=1 did not trace HTTP requests; stderr:\n%s", out)
	}
}

// TestEnvVarsAppearAsHelpDefaults is the `--help` half of H: env-provided
// defaults must be visible in flag help, because that is the only place
// users and agents can see which config is in effect.
func TestEnvVarsAppearAsHelpDefaults(t *testing.T) {
	bin := buildThrowawayBinary(t)

	out, err := runThrowawayBinary(t, bin, []string{
		"GITEE_TIMEOUT=42",
		"GITEE_HOSTNAME=enterprise.example.com",
		"GITEE_API_URL=http://127.0.0.1:9/api/v5",
		"GITEE_PAGINATE=1",
		"GITEE_NO_EXTENSIONS=1",
	}, "repo", "list", "--help")
	if err != nil {
		t.Fatalf("repo list --help failed: %v\n%s", err, out)
	}
	cases := []struct{ flag, want string }{
		{"--timeout", "(default 42)"},
		{"--hostname", "enterprise.example.com"},
		{"--api-url", "127.0.0.1:9"},
		{"--paginate", "(default true)"},
		{"--no-extensions", "(default true)"},
	}
	for _, tc := range cases {
		// Match the flag's own definition row: optional short form, then
		// the long flag. A plain substring match would also hit other
		// flags' descriptions (e.g. --page's text mentions --paginate).
		flagRE := regexp.MustCompile(`^\s*(-[A-Za-z], )?` + regexp.QuoteMeta(tc.flag) + `\b`)
		var line string
		for _, l := range strings.Split(out, "\n") {
			if flagRE.MatchString(l) {
				line = l
				break
			}
		}
		if line == "" {
			t.Errorf("flag %q not found in --help output", tc.flag)
			continue
		}
		if !strings.Contains(line, tc.want) {
			t.Errorf("%s default does not reflect env: got %q, want it to contain %q", tc.flag, line, tc.want)
		}
	}
}
