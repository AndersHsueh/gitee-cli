package types

// SSHKey represents a Gitee user SSH public key.
type SSHKey struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Key       string `json:"key"`
	CreatedAt string `json:"created_at"`
}

// SSHKeyCreateRequest is used when adding a new SSH public key.
type SSHKeyCreateRequest struct {
	Title string `json:"title"`
	Key   string `json:"key"`
}
