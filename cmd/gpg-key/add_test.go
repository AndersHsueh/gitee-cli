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

func TestValidateGPPublicKey(t *testing.T) {
	pub := "-----BEGIN PGP PUBLIC KEY BLOCK-----\n" +
		"Version: GnuPG v1\n\n" +
		"mQENBFxxxxx\n" +
		"-----END PGP PUBLIC KEY BLOCK-----\n"
	priv := "-----BEGIN PGP PRIVATE KEY BLOCK-----\n" +
		"Version: GnuPG v1\n\n" +
		"lQHYBFxxxxx\n" +
		"-----END PGP PRIVATE KEY BLOCK-----\n"

	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"public ok", pub, false},
		{"rejects private", priv, true},
		{"rejects empty", "", true},
		{"rejects random text", "not a pgp key", true},
		{"rejects missing begin marker", "-----END PGP PUBLIC KEY BLOCK-----\n", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateGPPublicKey(c.in)
			if (err != nil) != c.wantErr {
				t.Errorf("validateGPPublicKey(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
			}
		})
	}
}
