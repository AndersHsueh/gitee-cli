package help_topics

import (
	"bytes"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// runAndCapture executes cmd's RunE with an empty args slice and
// returns whatever was written to cmd's OutOrStdout (we redirect
// it to a buffer for the duration of the call). The topics use
// os.Stdout directly via tabwriter, so we also pipe stdout.
//
// cobra v1.10 has no public setter for the root command
// (SetRoot was removed), so we can't wire topics to a root
// from outside. We rely on the topic's Parent() chain, which
// the caller (test or main) sets up by adding the topic as a
// subcommand of a real root.
func runAndCapture(cmd *cobra.Command, root *cobra.Command) (string, error) {
	buf := &bytes.Buffer{}

	// Pipe os.Stdout so fmt.Fprintln(os.Stdout, ...) (used by
	// formatting.go for the jq recipes) is captured.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(buf, r)
		close(done)
	}()

	// Also redirect cobra's OutOrStdout for any topic that
	// uses cmd.Println / similar.
	cmd.SetOut(buf)
	root.SetOut(buf)

	err := cmd.RunE(cmd, nil)
	_ = w.Close()
	<-done
	return buf.String(), err
}
