package gpgkey

import (
	"strings"
	"testing"
)

func TestAddAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gpg-key add",
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
		if !strings.Contains(gpgKeyAddAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

func TestValidateGPGPublicKey(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"armored", "-----BEGIN PGP PUBLIC KEY BLOCK-----\n...rest...", false},
		{"empty", "", true},
		{"garbage", "not-a-key", true},
		{"private marker", "-----BEGIN PGP PRIVATE KEY BLOCK-----", true},
		{"missing prefix", "PGP PUBLIC KEY BLOCK-----", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateGPGPublicKey(c.in)
			if (err != nil) != c.wantErr {
				t.Errorf("validateGPGPublicKey(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
			}
		})
	}
}
