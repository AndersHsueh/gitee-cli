module gitee-cli

go 1.25.0

// Go 1.25.0's standard library is flagged by govulncheck (37 reachable
// findings, fixed by 1.26.10). The language version stays 1.25 so the
// module still builds with a 1.25-aware toolchain; the toolchain line
// selects a patched compiler for test, lint, and govulncheck.
toolchain go1.26.10

require (
	github.com/fatih/color v1.19.0
	github.com/go-resty/resty/v2 v2.17.2
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
