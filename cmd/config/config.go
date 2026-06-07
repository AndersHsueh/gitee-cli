package config

import (
	"encoding/json"
	"fmt"

	giteeconfig "gitee-cli/internal/config"

	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config <subcommand>",
	Short: "Manage gitee-cli configuration",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configuration values",
	RunE: func(c *cobra.Command, args []string) error {
		s, err := giteeconfig.Load()
		if err != nil {
			// No config yet — show empty
			fmt.Println("{}")
			return nil
		}
		b, _ := json.MarshalIndent(s.Preferences, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		s, err := giteeconfig.Load()
		if err != nil {
			return err
		}
		key := args[0]
		switch key {
		case "output":
			fmt.Println(s.Preferences.Output)
		case "editor":
			fmt.Println(s.Preferences.Editor)
		case "browser":
			fmt.Println(s.Preferences.Browser)
		case "pager":
			fmt.Println(s.Preferences.Pager)
		case "protocol":
			fmt.Println(s.Preferences.Protocol)
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}
		return nil
	},
}

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		s, err := giteeconfig.Load()
		if err != nil {
			s = &giteeconfig.Settings{
				Hosts:       make(map[string]giteeconfig.HostConfig),
				Preferences: giteeconfig.Preferences{},
			}
		}
		key, val := args[0], args[1]
		switch key {
		case "output":
			s.Preferences.Output = val
		case "editor":
			s.Preferences.Editor = val
		case "browser":
			s.Preferences.Browser = val
		case "pager":
			s.Preferences.Pager = val
		case "protocol":
			s.Preferences.Protocol = val
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}
		if err := giteeconfig.Save(s); err != nil {
			return err
		}
		fmt.Printf("Set %s = %s\n", key, val)
		return nil
	},
}

var unsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Unset a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		s, err := giteeconfig.Load()
		if err != nil {
			return err
		}
		key := args[0]
		switch key {
		case "output":
			s.Preferences.Output = ""
		case "editor":
			s.Preferences.Editor = ""
		case "browser":
			s.Preferences.Browser = ""
		case "pager":
			s.Preferences.Pager = ""
		case "protocol":
			s.Preferences.Protocol = ""
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}
		if err := giteeconfig.Save(s); err != nil {
			return err
		}
		fmt.Printf("Unset %s\n", key)
		return nil
	},
}

func init() {
	ConfigCmd.AddCommand(listCmd, getCmd, setCmd, unsetCmd)
}
