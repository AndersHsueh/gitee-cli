package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gitee-cli/cmd"
)

var rawFlags struct {
	method string
	fields []string
	raw    bool
}

var rawCmd = &cobra.Command{
	Use:   "raw <method> <path>",
	Short: "Make raw API request",
	Args:  cobra.ExactArgs(2),
	RunE: runRaw,
}

func init() {
	rawCmd.Flags().StringArrayVar(&rawFlags.fields, "field", []string{}, "Form field (key=value)")
	rawCmd.Flags().BoolVar(&rawFlags.raw, "raw", false, "Output raw response body")
}

func runRaw(cobraCmd *cobra.Command, args []string) error {
	method := strings.ToUpper(args[0])
	path := args[1]

	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	if !validMethods[method] {
		return fmt.Errorf("invalid method: %s (valid: GET, POST, PUT, PATCH, DELETE)", method)
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// Parse fields into body map
	var body map[string]interface{}
	if len(rawFlags.fields) > 0 && method != "GET" && method != "DELETE" {
		body = make(map[string]interface{})
		for _, f := range rawFlags.fields {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) == 2 {
				body[parts[0]] = parts[1]
			}
		}
	}

	var result json.RawMessage

	switch method {
	case "GET":
		err = client.Get(ctx, path, &result)
	case "POST":
		err = client.Post(ctx, path, body, &result)
	case "PATCH":
		err = client.Patch(ctx, path, body, &result)
	case "DELETE":
		err = client.Delete(ctx, path)
	case "PUT":
		err = client.Put(ctx, path, body, &result)
	}

	if err != nil {
		return err
	}

	if rawFlags.raw {
		io.WriteString(cobraCmd.OutOrStdout(), string(result))
	} else {
		var out bytes.Buffer
		json.Indent(&out, result, "", "  ")
		out.WriteTo(cobraCmd.OutOrStdout())
	}
	return nil
}
