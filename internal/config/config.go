package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Settings mirrors the structure of ~/.gitee-cli/settings.json
type Settings struct {
	Git struct {
		User struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"user"`
	} `json:"git"`
	Hosts       map[string]HostConfig `json:"hosts"`
	Preferences Preferences           `json:"preferences"`
}

type HostConfig struct {
	Token    string   `json:"token"`
	Username string   `json:"username"`
	Protocol string   `json:"protocol"`
	Scopes   []string `json:"scopes"`
}

type Preferences struct {
	Output   string `json:"output"`
	Editor   string `json:"editor"`
	Browser  string `json:"browser"`
	Prompt   string `json:"prompt"`
	Pager    string `json:"pager"`
	Protocol string `json:"protocol"`
}

func SettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gitee-cli", "settings.json")
}

func Load() (*Settings, error) {
	path := SettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Settings{Hosts: map[string]HostConfig{}}, nil
		}
		return nil, fmt.Errorf("reading settings: %w", err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}
	if s.Hosts == nil {
		s.Hosts = map[string]HostConfig{}
	}
	return &s, nil
}

func Save(s *Settings) error {
	path := SettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// GetToken returns the token for the given hostname.
// Priority: envvar GITEE_TOKEN > settings file
func GetToken(hostname string) string {
	if t := os.Getenv("GITEE_TOKEN"); t != "" {
		return t
	}
	s, err := Load()
	if err != nil {
		return ""
	}
	if hc, ok := s.Hosts[hostname]; ok {
		return hc.Token
	}
	return ""
}
