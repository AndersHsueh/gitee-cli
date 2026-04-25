package printer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

// OutputFormat represents the output format type.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
)

// Printer handles formatted output for CLI commands.
type Printer struct {
	Format OutputFormat
}

// NewPrinter creates a new Printer with the specified format.
func NewPrinter(format string) *Printer {
	p := &Printer{Format: FormatTable}
	switch format {
	case "json":
		p.Format = FormatJSON
	case "csv":
		p.Format = FormatCSV
	case "table":
		p.Format = FormatTable
	}
	return p
}

// PrintJSON prints data as formatted JSON.
func (p *Printer) PrintJSON(data interface{}) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// PrintJSONRaw prints raw JSON bytes.
func (p *Printer) PrintJSONRaw(data []byte) error {
	var out bytes.Buffer
	json.Indent(&out, data, "", "  ")
	out.WriteTo(os.Stdout)
	return nil
}

// TableWriter creates a new tabwriter for table output.
func (p *Printer) TableWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// PrintError prints an error message.
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// PrintSuccess prints a success message.
func PrintSuccess(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// PrintWarning prints a warning message.
func PrintWarning(format string, args ...interface{}) {
	fmt.Printf("Warning: "+format+"\n", args...)
}

// PrintInfo prints an info message.
func PrintInfo(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
