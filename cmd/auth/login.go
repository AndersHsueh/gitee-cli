package auth

import (
"fmt"

"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
Use:   "login",
Short: "Authenticate with a Gitee host",
Long: `Authenticate with gitee.com or a Gitee Enterprise instance.

The default authentication method is via a personal access token.`,
RunE: func(cobraCmd *cobra.Command, args []string) error {
fmt.Println("To authenticate, set your token via:")
fmt.Println("  export GITEE_TOKEN=<your-token>")
fmt.Println("  or set it in ~/.gitee-cli/settings.json")
return nil
},
}

func init() {
f := loginCmd.Flags()
f.StringP("token", "t", "", "Use a specific token for authentication")
f.StringSlice("scopes", nil, "Additional OAuth scopes to request")
f.String("hostname", "gitee.com", "The hostname of the Gitee instance to authenticate with")
f.Bool("web", false, "Open a browser to authenticate (not yet supported)")
}
