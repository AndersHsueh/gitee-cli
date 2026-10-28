package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gitee-cli/internal/atomicfile"
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
	Output     string `json:"output"`
	Editor     string `json:"editor"`
	Browser    string `json:"browser"`
	Prompt     string `json:"prompt"`
	Pager      string `json:"pager"`
	Protocol   string `json:"protocol"`
	ActiveHost string `json:"active_host"` // default host when --hostname not explicitly set (v0.10.0)
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
	// A settings file written by an older build (or copied/restored by hand)
	// may be group/world-readable even though it holds an access token.
	// Tighten it on read rather than waiting for the next Save. Best effort:
	// a read-only HOME or a filesystem that ignores chmod must not turn every
	// command into an authentication failure.
	if err := tightenPermissions(path); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
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

// tightenPermissions enforces the 0600 invariant on an existing settings file.
// Files can be group/world-readable because they predate the atomic-write
// behaviour, were restored from a backup, or were created by another tool —
// and the file contains an access token, so this is not cosmetic.
func tightenPermissions(path string) error {
	if runtime.GOOS == "windows" {
		// Windows has no POSIX group/world bits: Mode().Perm() is synthesized
		// as 0666 and os.Chmod only toggles the read-only flag, so the check
		// would fire on every load without ever converging.
		return nil
	}
	return atomicfile.EnsurePerm(path, 0o600)
}

func Save(s *Settings) error {
	path := SettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(path, data, 0o600)
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

// GetActiveHost returns the default hostname from preferences, defaulting
// to "gitee.com" if not configured. Used when --hostname is not explicitly
// provided (v0.10.0).
func GetActiveHost() string {
	s, err := Load()
	if err != nil || s.Preferences.ActiveHost == "" {
		return "gitee.com"
	}
	return s.Preferences.ActiveHost
}

// SetActiveHost updates the active host in preferences and saves settings
// atomically. The hostname must exist in the Hosts map (validated by the
// caller). (v0.10.0)
func SetActiveHost(hostname string) error {
	s, err := Load()
	if err != nil {
		return err
	}
	s.Preferences.ActiveHost = hostname
	return Save(s)
}
