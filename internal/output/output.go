package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

// NewTableWriter 返回统一配置的 tabwriter.Writer，所有表格输出使用相同格式。
func NewTableWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// PrintJSON marshals v as indented JSON and writes it to stdout. It is the
// shared JSON printer used across command and domain layers so the output
// shape never drifts.
func PrintJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
