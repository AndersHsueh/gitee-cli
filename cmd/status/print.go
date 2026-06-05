package status

import "fmt"

// fmtPrint is a thin indirection so that status_agenthelp.go does not
// need to import "fmt" directly. (Keeping the import in one place makes
// it easy to swap the output destination for tests — e.g. inject an
// io.Writer if we ever want to test the agent help text shape.)
func fmtPrint(s string) { fmt.Print(s) }
