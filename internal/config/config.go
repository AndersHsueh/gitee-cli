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
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
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
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	// 写到同目录的临时文件，再 rename 原子替换。
	// 同目录 rename 在 POSIX 和 Windows 上都是原子的，保证读者要么看到旧文件、
	// 要么看到完整的新文件，不会出现半截 JSON。
	tmp, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	// 任何出错都要清理掉临时文件，避免遗留垃圾。
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return err
	}
	return nil
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

// GetScopes returns the OAuth scopes for the given hostname.
func GetScopes(hostname string) []string {
	s, err := Load()
	if err != nil {
		return nil
	}
	if hc, ok := s.Hosts[hostname]; ok {
		return hc.Scopes
	}
	return nil
}

// GetUsername returns the username for the given hostname.
func GetUsername(hostname string) string {
	s, err := Load()
	if err != nil {
		return ""
	}
	if hc, ok := s.Hosts[hostname]; ok {
		return hc.Username
	}
	return ""
}

// GetAllHosts returns all configured hosts.
func GetAllHosts() []string {
	s, err := Load()
	if err != nil {
		return nil
	}
	hosts := make([]string, 0, len(s.Hosts))
	for h := range s.Hosts {
		hosts = append(hosts, h)
	}
	return hosts
}
