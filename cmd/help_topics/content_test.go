package help_topics

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"text/tabwriter"

	"gitee-cli/internal/envconfig"

	"github.com/spf13/cobra"
)

// freshRoot builds a throwaway cobra root + registers the help
// topics against it. Used by the content tests so they don't
// depend on the live rootCmd (which is package-private in
// gitee-cli/cmd and would create an import cycle if imported).
func freshRoot() *cobra.Command {
	root := &cobra.Command{Use: "gitee"}
	RegisterHelpTopics(root)
	return root
}

// runTopic executes topic's RunE with no args and returns the
// captured stdout. The topic's parent (root) is wired in so the
// reference topic can walk the tree.
func runTopic(t *testing.T, root *cobra.Command, topic *cobra.Command) string {
	t.Helper()
	out, err := runAndCapture(topic, root)
	if err != nil {
		t.Fatalf("topic RunE failed: %v", err)
	}
	return out
}

// TestEnvironmentTopicContent spot-checks the rendered table. The full
// documented set is verified by TestEnvironmentTopicListsExactlyImplementedVars
// against the shared registry, so this test only needs a couple of anchors.
func TestEnvironmentTopicContent(t *testing.T) {
	root := freshRoot()
	out := runTopic(t, root, environmentTopic)
	for _, want := range []string{"GITEE_TOKEN", "GITEE_HOSTNAME", "GITEE_OUTPUT", "NO_COLOR", "BROWSER"} {
		if !strings.Contains(out, want) {
			t.Errorf("environment topic missing %q\n--- got ---\n%s", want, out)
		}
	}
}

// TestEnvironmentTopicListsExactlyImplementedVars is the honesty test the
// environment page was missing: it used to document GITEE_INSECURE,
// GITEE_EDITOR, GITEE_BROWSER and CLICOLOR_FORCE — none of which the binary
// ever read.
//
// There is no second hand-written list here: the documented rows are derived
// from internal/envconfig.Variables (the same registry the runtime env layer
// consumes), and this test asserts the projection is lossless. Deleting a
// registry row therefore changes this page, and is caught by envconfig's
// TestCanonicalVariableSet.
func TestEnvironmentTopicListsExactlyImplementedVars(t *testing.T) {
	documented := environmentVars()
	registry := envconfig.Variables

	if len(documented) != len(registry) {
		t.Fatalf("environment topic renders %d variables, registry has %d", len(documented), len(registry))
	}
	for i, v := range registry {
		want := envVar{Name: v.Name, Default: v.Default, Description: v.Description}
		if documented[i] != want {
			t.Errorf("documented[%d] = %+v, want %+v (environment table must derive from envconfig.Variables)", i, documented[i], want)
		}
	}
}

func TestFormattingTopicContent(t *testing.T) {
	root := freshRoot()
	out := runTopic(t, root, formattingTopic)
	required := []string{"table", "json", "jq"}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("formatting topic missing reference to %q", want)
		}
	}
	// csv was never implemented; -o csv now errors, so the formatting page
	// must not advertise it as a format.
	if strings.Contains(out, "csv") {
		t.Errorf("formatting topic still advertises csv (never implemented, now rejected):\n%s", out)
	}
}

func TestReferenceTopicContent(t *testing.T) {
	// referenceTopic.Root() returns the help command (not the
	// root) when run in isolation — cobra's SetHelpCommand
	// doesn't wire a parent. Test walkCommands directly against
	// a synthetic tree.
	fakeRoot := &cobra.Command{Use: "gitee"}
	fakeRoot.AddCommand(&cobra.Command{Use: "fake", Short: "fake for test"})
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COMMAND\tSHORT")
	walkCommands(w, fakeRoot, "")
	if err := w.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if !strings.Contains(buf.String(), "fake") {
		t.Errorf("walkCommands should print fake; got:\n%s", buf.String())
	}
}

// TestReferenceTopicIncludesVersion locks the fix for a real gap: the
// reference walked the tree but skipped `version`, so the auto-generated
// index omitted a command the binary actually ships. Every non-hidden
// command must appear — especially version, which is also the one users
// are most likely to look up.
func TestReferenceTopicIncludesVersion(t *testing.T) {
	fakeRoot := &cobra.Command{Use: "gitee"}
	fakeRoot.AddCommand(&cobra.Command{Use: "version", Short: "Print the version of gitee-cli"})
	fakeRoot.AddCommand(&cobra.Command{Use: "hidden-thing", Short: "hidden", Hidden: true})
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COMMAND\tSHORT")
	walkCommands(w, fakeRoot, "")
	if err := w.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "gitee version") {
		t.Errorf("walkCommands must list 'gitee version'; got:\n%s", out)
	}
	if strings.Contains(out, "hidden-thing") {
		t.Errorf("walkCommands must not list hidden commands; got:\n%s", out)
	}
}

func TestRegisterHelpTopicsWiring(t *testing.T) {
	root := &cobra.Command{Use: "gitee"}
	help := RegisterHelpTopics(root)
	if help == nil {
		t.Fatal("RegisterHelpTopics should return the help command it installed")
	}
	got := map[string]bool{}
	for _, c := range help.Commands() {
		got[c.Name()] = true
	}
	for _, want := range []string{"environment", "exit-codes", "formatting", "reference"} {
		if !got[want] {
			t.Errorf("help command missing topic %q", want)
		}
	}
}

func TestRegisterHelpTopicsIdempotent(t *testing.T) {
	// Calling twice should not panic. SetHelpCommand replaces
	// the help command wholesale, so the test mainly verifies
	// the second call doesn't crash.
	root := &cobra.Command{Use: "gitee"}
	RegisterHelpTopics(root)
	help := RegisterHelpTopics(root)
	if help == nil {
		t.Fatal("help command gone after double register")
	}
}
