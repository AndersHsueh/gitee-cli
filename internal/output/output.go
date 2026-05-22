package output

import (
	"os"
	"text/tabwriter"
)

// NewTableWriter 返回统一配置的 tabwriter.Writer，所有表格输出使用相同格式。
func NewTableWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}