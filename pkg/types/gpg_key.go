package types

// GPGKey represents a Gitee user GPG public key.
type GPGKey struct {
	ID              int64         `json:"id"`
	PrimaryKeyID    string        `json:"primary_key_id"`
	PrimaryKeyEmail string        `json:"primary_key_email"`
	Emails          []GPGKeyEmail `json:"emails"`
	Subkeys         []GPGSubkey   `json:"subkeys"`
	RawKey          string        `json:"raw_key"` // ASCII-armored public key block
	CreatedAt       string        `json:"created_at"`
	ExpiresAt       string        `json:"expires_at,omitempty"`
}

// GPGKeyEmail represents an email bound to a GPG key.
type GPGKeyEmail struct {
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
}

// GPGSubkey represents a subkey of a GPG key.
type GPGSubkey struct {
	ID     int64         `json:"id"`
	KeyID  string        `json:"key_id"`
	Emails []GPGKeyEmail `json:"emails"`
}

// GPGKeyCreateRequest is used when adding a new GPG public key.
// The field name follows the Gitee v5 API: "armored_public_key".
type GPGKeyCreateRequest struct {
	ArmoredPublicKey string `json:"armored_public_key"`
}
