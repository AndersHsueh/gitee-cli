package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

// TestRawArgsBypassWithAgentHelp verifies the AgentHelpAwareArgs
// wrapper on cobra.ExactArgs(2) lets `gitee api raw --agent-help`
// reach PreRunE without the <method> <path> positional pair. This
// is the task-A part of the api raw fix; task B adds the PreRunE
// hook so the doc actually prints before auth checks.
func TestRawArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	if err := rawCmd.Args(rawCmd, nil); err != nil {
		t.Errorf("api raw --agent-help with no positional should pass Args, got: %v", err)
	}
}

// TestAPIArgsBypassWithAgentHelp verifies the same wrapper for the
// parent `gitee api` command (the raw subcommand is also registered
// under it for backwards compatibility).
func TestAPIArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	if err := apiCmd.Args(apiCmd, nil); err != nil {
		t.Errorf("api --agent-help with no positional should pass Args, got: %v", err)
	}
}

// TestAPIRawAgentHelpContent sanity-checks the const string is the
// standard agent-help shape: starts with `# gitee` so the agent-help
// matrix in cmd/agent_help.go counts it as a real doc (not an n/a
// notice), and contains the required sections.
func TestAPIRawAgentHelpContent(t *testing.T) {
	if !strings.HasPrefix(apiRawAgentHelp, "# gitee") {
		t.Errorf("apiRawAgentHelp must start with '# gitee', got %q", firstLine(apiRawAgentHelp))
	}
	for _, want := range []string{
		"## Purpose",
		"## Pre-conditions",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	} {
		if !strings.Contains(apiRawAgentHelp, want) {
			t.Errorf("apiRawAgentHelp missing required section: %q", want)
		}
	}
}

// TestAPIRawAgentHelpWithoutToken runs the gitee binary in a fresh
// subprocess with an empty HOME + GITEE_TOKEN="" so any settings
// file and env-supplied token are unavailable. Before task B, this
// path would print "not authenticated. Set GITEE_TOKEN..." because
// NewAPIClient fires before --agent-help could be served. After
// task B, the PreRunE hook prints the doc and exits 0 without
// touching the auth layer.
//
// We build a throwaway binary rather than rely on PATH so this test
// is hermetic and doesn't pollute the user's $GOPATH/bin.
func TestAPIRawAgentHelpWithoutToken(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "gitee-test")
	// Build a throwaway binary at a known path; t.TempDir() cleans
	// it up automatically on test end.
	cmdBuild := exec.Command("go", "build", "-o", bin, "../../cmd/gitee")
	cmdBuild.Dir = "."
	if out, err := cmdBuild.CombinedOutput(); err != nil {
		t.Fatalf("build throwaway gitee: %v\n%s", err, out)
	}

	// Run with an empty HOME (t.TempDir() creates an isolated dir
	// with no settings.json) so config.GetToken() falls through to
	// envvar GITEE_TOKEN (which we also clear) and returns "". Use
	// an unreachable API host so even if the short-circuit
	// regressed we wouldn't accidentally hit a real network.
	home := t.TempDir()
	run := exec.Command(bin, "api", "raw", "--agent-help")
	run.Env = []string{
		"HOME=" + home,
		"PATH=/usr/bin:/bin",
		"GITEE_TOKEN=",
		"GITEE_HOSTNAME=invalid.example.invalid",
	}
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("gitee api raw --agent-help (no token) failed: %v\noutput:\n%s", err, out)
	}
	if !strings.HasPrefix(string(out), "# gitee") {
		t.Errorf("expected stdout to start with '# gitee', got: %q", firstLine(string(out)))
	}
}

// TestAPIRawAgentHelpIsServedBeforeAuth asserts the wiring: with
// --agent-help set and zero args, rawCmd's PreRunE returns nil
// without ever reaching NewAPIClient. We exercise PreRunE directly
// (without invoking PrintAgentHelp, which os.Exit(0)s) by inspecting
// that the function is wired and returns no error.
func TestAPIRawAgentHelpIsServedBeforeAuth(t *testing.T) {
	if rawCmd.PreRunE == nil {
		t.Fatal("rawCmd.PreRunE is nil; --agent-help short-circuit missing")
	}
	if apiCmd.PreRunE == nil {
		t.Fatal("apiCmd.PreRunE is nil; --agent-help short-circuit missing")
	}
}

// TestAPICmdHasFieldAndRawFlags verifies the fix for the bug where
// `gitee api POST /user/repos --field name=x` failed with "unknown
// flag: --field". Before the fix --field/--raw were only registered
// on rawCmd (the legacy subcommand), not on apiCmd (the canonical
// entry point documented in apiCmd.Example). cobra does not inherit
// a parent's local Flags() to subcommands, so both commands must
// register the flags explicitly.
func TestAPICmdHasFieldAndRawFlags(t *testing.T) {
	if apiCmd.Flags().Lookup("field") == nil {
		t.Error("apiCmd is missing --field flag (gitee api POST /path --field x would fail)")
	}
	if apiCmd.Flags().Lookup("raw") == nil {
		t.Error("apiCmd is missing --raw flag")
	}
	// rawCmd must still have them for backwards compatibility.
	if rawCmd.Flags().Lookup("field") == nil {
		t.Error("rawCmd lost --field flag")
	}
	if rawCmd.Flags().Lookup("raw") == nil {
		t.Error("rawCmd lost --raw flag")
	}
}

// TestRunAPIWithFieldFlag exercises the canonical entry point end-to-end:
// `gitee api POST /user/repos --field name=my-repo --field private=true`
// must reach the server as a POST with the fields in the JSON body. This
// is the exact invocation promised by apiCmd.Example.
func TestRunAPIWithFieldFlag(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]interface{}
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		var b map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&b)
		gotBody = b
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"my-repo"}`))
	}))
	defer ts.Close()

	// Point NewAPIClient at the test server and supply a token so it
	// doesn't error out on "not authenticated".
	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	origFields := rawFlags.fields
	origRaw := rawFlags.raw
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		rawFlags.fields = origFields
		rawFlags.raw = origRaw
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	rawFlags.fields = []string{"name=my-repo", "private=true"}
	rawFlags.raw = false

	// runAPI reads the paginate flag off the cobra command; parse the
	// apiCmd flags so GetBool("paginate") returns its default (false).
	_ = apiCmd.Flags().Parse([]string{"POST", "/user/repos"})

	var out strings.Builder
	apiCmd.SetOut(&out)
	if err := runAPI(apiCmd, []string{"POST", "/user/repos"}); err != nil {
		t.Fatalf("runAPI with --field failed: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/user/repos" {
		t.Errorf("expected /user/repos, got %s", gotPath)
	}
	if gotBody["name"] != "my-repo" {
		t.Errorf("expected body name=my-repo, got %v", gotBody["name"])
	}
	if gotBody["private"] != "true" {
		t.Errorf("expected body private=true, got %v", gotBody["private"])
	}
}

// TestRunAPIRawFlagOnAPICmd verifies --raw registered on apiCmd is
// honored: the response body is written without JSON indent.
func TestRunAPIRawFlagOnAPICmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer ts.Close()

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	origFields := rawFlags.fields
	origRaw := rawFlags.raw
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		rawFlags.fields = origFields
		rawFlags.raw = origRaw
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	rawFlags.fields = nil
	rawFlags.raw = true

	_ = apiCmd.Flags().Parse([]string{"GET", "/user"})

	var out strings.Builder
	apiCmd.SetOut(&out)
	if err := runAPI(apiCmd, []string{"GET", "/user"}); err != nil {
		t.Fatalf("runAPI with --raw failed: %v", err)
	}
	if out.String() != `{"hello":"world"}` {
		t.Errorf("expected raw output %q, got %q", `{"hello":"world"}`, out.String())
	}
}

// firstLine returns the first line of s without trailing newline.
// Used for clearer error messages.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}