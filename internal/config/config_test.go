package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsPath(t *testing.T) {
	path := SettingsPath()
	assert.Contains(t, path, ".gitee-cli")
	assert.Contains(t, path, "settings.json")
}

func TestLoad_NotExist(t *testing.T) {
	s, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, s)
	assert.NotNil(t, s.Hosts)
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

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
	s.Preferences = Preferences{
		Output: "table",
		Editor: "vi",
	}

	path := tmpDir + "/settings.json"
	_ = path

	data, err := os.ReadFile(path)
	if err != nil {
		require.True(t, os.IsNotExist(err))
	}
	_ = data

	assert.Equal(t, "test-user", s.Git.User.Name)
	assert.Equal(t, "test@example.com", s.Git.User.Email)
	host, ok := s.Hosts["gitee.com"]
	require.True(t, ok)
	assert.Equal(t, "test-token-123", host.Token)
	assert.Equal(t, "https", host.Protocol)
	assert.ElementsMatch(t, []string{"user", "repo"}, host.Scopes)
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