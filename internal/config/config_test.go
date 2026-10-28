package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withHome points $HOME (and %USERPROFILE% for Windows) at a temp dir so the
// test never reads or writes the developer's real ~/.gitee-cli/settings.json.
func withHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

func TestSettingsPath(t *testing.T) {
	path := SettingsPath()
	assert.Contains(t, path, ".gitee-cli")
	assert.Contains(t, path, "settings.json")
}

func TestLoad_NotExist(t *testing.T) {
	withHome(t)

	s, err := Load()
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.NotNil(t, s.Hosts)
}

func TestLoad_InvalidJSON(t *testing.T) {
	withHome(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(SettingsPath()), 0o700))
	require.NoError(t, os.WriteFile(SettingsPath(), []byte("{not json"), 0o600))

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing settings")
}

// TestSaveAndLoadRoundTrip is the real round-trip test: it exercises Save()
// then Load() against an isolated HOME and asserts both the payload and the
// 0600/0700 permission invariants.
func TestSaveAndLoadRoundTrip(t *testing.T) {
	withHome(t)

	s := &Settings{}
	s.Git.User.Name = "test-user"
	s.Git.User.Email = "test@example.com"
	s.Hosts = map[string]HostConfig{
		"gitee.com": {
			Token:    "test-token-123",
			Username: "test-user",
			Protocol: "https",
			Scopes:   []string{"user", "repo"},
		},
	}
	s.Preferences = Preferences{Output: "table", Editor: "vi"}

	require.NoError(t, Save(s))

	path := SettingsPath()
	fileInfo, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm(), "settings file must be 0600")

	dirInfo, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm(), "settings dir must be 0700")

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "test-user", loaded.Git.User.Name)
	assert.Equal(t, "test@example.com", loaded.Git.User.Email)
	host, ok := loaded.Hosts["gitee.com"]
	require.True(t, ok)
	assert.Equal(t, "test-token-123", host.Token)
	assert.Equal(t, "https", host.Protocol)
	assert.ElementsMatch(t, []string{"user", "repo"}, host.Scopes)
	assert.Equal(t, "table", loaded.Preferences.Output)
}

// TestLoad_TightensLoosePermissions covers the field case that triggered this
// fix: an existing settings.json was 0644 while holding a live token, and
// nothing ever repaired it because Load() did not check.
func TestLoad_TightensLoosePermissions(t *testing.T) {
	withHome(t)

	dir := filepath.Dir(SettingsPath())
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(SettingsPath(), []byte(`{
		"hosts": {"gitee.com": {"token": "secret-token", "username": "u"}}
	}`), 0o644))
	require.NoError(t, os.Chmod(SettingsPath(), 0o644))

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "secret-token", loaded.Hosts["gitee.com"].Token)

	info, err := os.Stat(SettingsPath())
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "Load must tighten a loose settings file to 0600")
}

func TestDefaultPreferences(t *testing.T) {
	s := &Settings{
		Hosts: map[string]HostConfig{},
	}
	assert.Empty(t, s.Preferences.Output)
	assert.Empty(t, s.Preferences.Editor)
}

func TestHostConfig(t *testing.T) {
	hc := HostConfig{
		Token:    "token",
		Username: "user",
		Protocol: "https",
		Scopes:   []string{"repo"},
	}
	assert.Equal(t, "token", hc.Token)
	assert.Equal(t, "user", hc.Username)
	assert.Equal(t, "https", hc.Protocol)
}
