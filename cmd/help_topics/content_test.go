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

func TestEnvironmentTopicContent(t *testing.T) {
	root := freshRoot()
	out := runTopic(t, root, environmentTopic)
	required := []string{
		"GITEE_TOKEN",
		"NO_COLOR",
		"HTTPS_PROXY",
		"GITEE_EXT_DIR",
		"GITEE_ALIASES_FILE",
		"BROWSER",
		"Not read (setting these has no effect):",
		"GITEE_PAGINATE",
		"GITEE_HOSTNAME",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("environment topic missing %q\n--- got ---\n%s", want, out)
		}
	}
	// The "not read" disclaimer may name these. The variable table
	// (everything before that heading) must not list them as read.
	table, _, _ := strings.Cut(out, "Not read")
	for _, name := range []string{"GITEE_PAGINATE", "GITEE_OUTPUT", "GITEE_HOSTNAME"} {
		for _, line := range strings.Split(table, "\n") {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] == name {
				t.Errorf("%s listed as a read variable: %q", name, line)
			}
		}
	}
}

func TestExitCodesTopicContent(t *testing.T) {
	root := freshRoot()
	out := runTopic(t, root, exitCodesTopic)
	required := []string{"0", "1", "2", "3", "4", "5", "64", "130"}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("exit-codes topic missing code %q", want)
		}
	}
}

func TestFormattingTopicContent(t *testing.T) {
	root := freshRoot()
	out := runTopic(t, root, formattingTopic)
	required := []string{"table", "json", "jq", "html_url", "not implemented", "not NDJSON", "GITEE_OUTPUT is not read"}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("formatting topic missing reference to %q\n--- got ---\n%s", want, out)
		}
	}
	if strings.Contains(out, "RFC 4180") {
		t.Error("formatting topic still describes csv as an implemented RFC 4180 format")
	}
	if strings.Contains(out, "GITEE_OUTPUT") && !strings.Contains(out, "GITEE_OUTPUT is not read") {
		t.Error("formatting topic must say GITEE_OUTPUT is not read")
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
