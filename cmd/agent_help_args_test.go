package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestAgentHelpAwareArgs_BypassesWhenAgentHelp verifies the wrapper
// returns nil when --agent-help is set, regardless of the inner
// validator's verdict. This is the core fix for the 17 commands that
// used to error out at ValidateArgs before PreRunE could print help.
func TestAgentHelpAwareArgs_BypassesWhenAgentHelp(t *testing.T) {
	orig := GlobalAgentHelp
	defer func() { GlobalAgentHelp = orig }()
	GlobalAgentHelp = true

	called := false
	wrapped := AgentHelpAwareArgs(func(c *cobra.Command, args []string) error {
		called = true
		return errors.New("inner should not be called when --agent-help is set")
	})

	cmd := &cobra.Command{Use: "fake"}
	if err := wrapped(cmd, nil); err != nil {
		t.Errorf("wrapper returned error with --agent-help set: %v", err)
	}
	if called {
		t.Error("inner validator should not be called when --agent-help is set")
	}
}

// TestAgentHelpAwareArgs_DelegatesWhenAgentHelpOff verifies the wrapper
// falls through to the inner validator when --agent-help is not set,
// so normal arg validation still works.
func TestAgentHelpAwareArgs_DelegatesWhenAgentHelpOff(t *testing.T) {
	orig := GlobalAgentHelp
	defer func() { GlobalAgentHelp = orig }()
	GlobalAgentHelp = false

	wantErr := errors.New("accepts 1 arg(s), received 0")
	wrapped := AgentHelpAwareArgs(func(c *cobra.Command, args []string) error {
		return wantErr
	})

	c := &cobra.Command{Use: "fake"}
	got := wrapped(c, nil)
	if got == nil {
		t.Fatal("expected inner validator error to surface; got nil")
	}
	if !strings.Contains(got.Error(), "accepts 1 arg") {
		t.Errorf("got %q, want it to propagate inner error", got.Error())
	}
}

// TestAgentHelpAwareArgs_PassesWhenInnerPasses verifies the wrapper
// does not interfere when --agent-help is off AND the inner validator
// accepts the args.
func TestAgentHelpAwareArgs_PassesWhenInnerPasses(t *testing.T) {
	orig := GlobalAgentHelp
	defer func() { GlobalAgentHelp = orig }()
	GlobalAgentHelp = false

	wrapped := AgentHelpAwareArgs(func(c *cobra.Command, args []string) error {
		return nil
	})
	c := &cobra.Command{Use: "fake"}
	if err := wrapped(c, []string{"a", "b"}); err != nil {
		t.Errorf("wrapper returned error on valid args: %v", err)
	}
}
