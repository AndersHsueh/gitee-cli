package sshkey

import (
	"strings"
	"testing"
)

func TestAddAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee ssh-key add",
		"Purpose",
		"Pre-conditions",
		"Required flags",
		"Idempotency",
		"NOT idempotent",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"OAuth scope required",
		"Rate limit",
		"Next steps",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(sshKeyAddAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

func TestValidateSSHPublicKey(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"rsa", "ssh-rsa AAAA... user@host", false},
		{"ed25519", "ssh-ed25519 AAAA... user@host", false},
		{"ecdsa 256", "ecdsa-sha2-nistp256 AAAA... user@host", false},
		{"ecdsa 384", "ecdsa-sha2-nistp384 AAAA... user@host", false},
		{"ecdsa 521", "ecdsa-sha2-nistp521 AAAA... user@host", false},
		{"dss", "ssh-dss AAAA... user@host", false},
		{"empty", "", true},
		{"garbage", "not-a-key", true},
		{"private marker", "-----BEGIN RSA PRIVATE KEY-----", true},
		{"missing space", "ssh-rsaAAAA", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateSSHPublicKey(c.in)
			if (err != nil) != c.wantErr {
				t.Errorf("validateSSHPublicKey(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
			}
		})
	}
}
