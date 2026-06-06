package help_topics

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// freshRoot builds a throwaway cobra root + registers the help
// topics against it. Used by the wiring tests so they don't
// depend on the live rootCmd (which is package-private in the
// gitee-cli/cmd package and would create an import cycle if we
// tried to import it).
func freshRoot() *RootAlias {
	root := &cobra.Command{Use: "gitee"}
	RegisterHelpTopics(root)
	return &RootAlias{root: root}
}

// RootAlias is a thin shim that exposes the root and help
// command we created in freshRoot(). We can't make RootCommand
// and HelpCommandForTest a single method on the test type without
// dragging the package's exported types into the alias, but
// this is good enough.
type RootAlias struct {
	root *cobra.Command
}

// runTopic executes topic's RunE with no args and returns the
// captured stdout. The topic's parent (root) is wired in so the
// reference topic can walk the tree.
func (r *RootAlias) runTopic(t *testing.T, topic *cobra.Command) string {
	t.Helper()
	out, err := runAndCapture(topic, r.root)
	if err != nil {
		t.Fatalf("topic RunE failed: %v", err)
	}
	return out
}

func TestEnvironmentTopicContent(t *testing.T) {
	r := freshRoot()
	out := r.runTopic(t, environmentTopic)
	required := []string{
		"GITEE_TOKEN",
		"GITEE_HOSTNAME",
		"GITEE_API_URL",
		"GITEE_OUTPUT",
		"NO_COLOR",
		"GITEE_PAGINATE",
		"GITEE_EXT_DIR",
		"GITEE_ALIASES_FILE",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("environment topic missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestExitCodesTopicContent(t *testing.T) {
	r := freshRoot()
	out := r.runTopic(t, exitCodesTopic)
	required := []string{"0", "1", "2", "3", "4", "5", "64", "130"}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("exit-codes topic missing code %q", want)
		}
	}
}

func TestFormattingTopicContent(t *testing.T) {
	r := freshRoot()
	out := r.runTopic(t, formattingTopic)
	required := []string{"table", "json", "csv", "jq"}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("formatting topic missing reference to %q", want)
		}
	}
}

func TestReferenceTopicContent(t *testing.T) {
	// We don't exercise the live cobra tree here because
	// referenceTopic.Root() returns the help command (not the
	// root) when the topic is run in isolation — cobra's
	// SetHelpCommand doesn't wire a parent. The topic's logic
	// is therefore tested by hand-running walkCommands against
	// a synthetic tree.
	r := freshRoot()
	fakeRoot := &cobra.Command{Use: "gitee"}
	fakeRoot.AddCommand(&cobra.Command{Use: "fake", Short: "fake for test"})
	// Walk fakeRoot and verify the walker prints "fake".
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COMMAND\tSHORT")
	walkCommands(w, fakeRoot, "", map[string]bool{})
	if err := w.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if !strings.Contains(buf.String(), "fake") {
		t.Errorf("walkCommands should print fake; got:\n%s", buf.String())
	}
	_ = r
}

func TestRegisterHelpTopicsWiring(t *testing.T) {
	freshRoot()
	help := HelpCommandForTest()
	if help == nil {
		t.Fatal("after RegisterHelpTopics, HelpCommandForTest() should not be nil")
	}
	got := map[string]bool{}
	for _, c := range help.Commands() {
		got[c.Name()] = true
	}
	for _, want := range AllTopicNames() {
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
	RegisterHelpTopics(root) // must not panic
	if HelpCommandForTest() == nil {
		t.Fatal("help command gone after double register")
	}
}
