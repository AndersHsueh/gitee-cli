# gitee-cli

[中文](README.md) | **English** | [日本語](README_ja.md)

> Gitee Official CLI Tool — Complete Gitee operations in the terminal, equivalent to GitHub's `gh`

[![Go Version](https://img.shields.io/badge/go-1.27+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.13.2-orange.svg)](https://gitee.com/andershsueh/gitee-cli/releases/latest)

---

## Features

```
gitee auth        # Authentication (login / logout / status / token)               ✅ Implemented
gitee issue       # Issue Management (list / create / view / edit / close / reopen / comment) ✅ Implemented
gitee config      # Configuration (set / get / list / unset)                        ✅ Implemented
gitee search      # Search issues / users (repos currently empty upstream; code/commits unsupported) ⚠️ Partial
gitee completion  # Shell Completion (bash / zsh / fish / powershell)                ✅ Implemented
gitee version     # Version Info                                                    ✅ Implemented
gitee repo        # Repo Management (list / create / view / fork / edit / delete / clone / browse / branches / star / watch / protect / file / commits / compare) ✅ Implemented
gitee pr          # Pull Request (list / create / view / merge / review / comments / diff) ✅ Implemented
gitee gist        # Gist (list / create / view / edit / delete / star / unstar)     ✅ Implemented
gitee org         # Organization (list / view / members / repos / issues)           ✅ Implemented
gitee api         # Direct Gitee API v5 calls (GET / POST / PUT / PATCH / DELETE)   ✅ Implemented
gitee --agent-help # Structured, agent-targeted help (11 dimensions per command)    ✅ Implemented
```

> `search` reality check (v0.13.2): `gitee search issues` / `gitee search users` work;
> `gitee search repos` currently returns no results for every query (upstream API behavior)
> and prints `(no results)`; `gitee search code` / `gitee search commits` return 404 on
> this Gitee instance — the CLI explains this and exits non-zero instead of pretending success.

> 📖 The full `--help` / `--agent-help` dual-surface design rationale lives in the
> Chinese [README.md](README.md) and [docs/philosophy.md](docs/philosophy.md).

---

## Quick Start

### Build from Source

```bash
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go install ./cmd/gitee    # installs to $(go env GOPATH)/bin
```

### Authentication

Configure Token in `~/.gitee-cli/settings.json` (see `settings.json.sample`):

```json
{
  "hosts": {
    "gitee.com": {
      "token": "your_access_token",
      "username": "your_username"
    }
  }
}
```

Or use environment variable:

```bash
export GITEE_TOKEN=your_access_token
```

---

## Installation

**Binary Download**

Download release archives from [Releases](https://gitee.com/andershsueh/gitee-cli/releases/latest):

| Platform | Filename |
|----------|----------|
| Linux amd64 | `gitee-linux-amd64.tar.gz` |
| Linux arm64 | `gitee-linux-arm64.tar.gz` |
| macOS amd64 | `gitee-darwin-amd64.tar.gz` |
| macOS arm64 (M1/M2) | `gitee-darwin-arm64.tar.gz` |
| Windows amd64 | `gitee-windows-amd64.exe.zip` |

Each archive contains the binary and `LICENSE`; `SHA256SUMS.txt` is published alongside
them (`sha256sum -c SHA256SUMS.txt` on Linux/macOS, `certutil -hashfile <file> SHA256` on Windows).

**Build from Source**

```bash
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go install ./cmd/gitee    # installs to $(go env GOPATH)/bin
```

### Authentication

`gitee auth login` does **not** run an OAuth device flow or open a browser. It reads an
existing personal access token (from `GITEE_TOKEN` or `~/.gitee-cli/settings.json`) and
verifies it against `/user`:

```bash
export GITEE_TOKEN=your_access_token   # recommended: not written to disk
gitee auth status                      # verify the token

# or persist a token to ~/.gitee-cli/settings.json (0600)
gitee auth login --token your_access_token
```

`gitee auth refresh` is currently a stub: it prints that token refresh is not supported and
tells you to run `gitee auth login` again. It does not refresh anything and is **not** an OAuth
`refresh_token` exchange.

> ⚠️ A `--token` value is visible to other local users via `ps` and lands in shell history;
> prefer `GITEE_TOKEN` outside one-off scripts.

---

## Usage Examples

### Repository Management

```bash
# List my repos
gitee repo list

# List public repos of a user
gitee repo list --owner torvalds --visibility public

# Create repo
gitee repo create --name my-project --description "My new project" --private

# Clone repo
gitee repo clone owner/repo-name

# Fork repo
gitee repo fork owner/repo-name

# Open repo in browser
gitee repo browse owner/repo-name
```

### Issue Management

```bash
# List open issues in current repo
gitee issue list

# List all states, filter by labels
gitee issue list --state all --labels bug,enhancement

# Create issue
gitee issue create --title "Found a Bug" --body "Steps to reproduce:..." --labels bug

# View issue
gitee issue view 42

# Close issue
gitee issue close 42

# Add comment
gitee issue comment 42 --body "Thanks for reporting, fixed now"
```

### Pull Request

```bash
# List all open PRs
gitee pr list

# Create PR
gitee pr create --title "feat: add search functionality" --source feature/search --target main

# View PR diff
gitee pr diff 18

# Merge PR (squash)
gitee pr merge 18 --squash

# Review PR
gitee pr review 18 --approve
gitee pr review 18 --request-changes --body "Please add unit tests"
```

### Search

```bash
# Search issues (works)
gitee search issues "memory leak" --state open

# Search users (works)
gitee search users "john"

# ⚠️ Upstream currently returns no results for repository search
gitee search repos "web framework" --language go --sort stars   # ⚠️ currently returns no results

# ⚠️ Code / commit search is NOT supported on this Gitee instance
gitee search code "func main" --repo owner/repo   # ⚠️ 404 here, not supported (CLI explains, exits non-zero)
```

### Raw API Access

```bash
# GET request
gitee api GET /user
gitee api GET /user/repos?type=owner&per_page=10

# POST request (--field for params)
gitee api POST /user/repos --field name=test --field private=true

# Paginate through all results
gitee api GET /user/repos --paginate

# Inspect HTTP traffic
gitee api GET /user --verbose
```

### Configuration

```bash
gitee config set editor vim         # Set editor
gitee config set protocol ssh       # Use SSH for clone
gitee config get protocol           # Read config
gitee config list                   # View all config
gitee config unset editor           # Delete config item
```

---

## Global Options

| Option | Description | Default |
|--------|-------------|---------|
| `-R, --repo string` | Target repository (owner/repo); inferred from git remote when omitted | auto |
| `--token string` | Access Token (overrides config); prefer `GITEE_TOKEN` | from config |
| `--hostname string` | Gitee instance address (enterprise) | `gitee.com` |
| `--api-url string` | API base URL (non-standard private deployments) | `https://{hostname}/api/v5` |
| `-o, --output string` | Output format: `table` \| `json` (no CSV; invalid values error out) | `table` |
| `-P, --paginate` | Auto-paginate, fetch all results | `false` |
| `--limit int` | Max results (total cap across pages with `--paginate`; `0` = unlimited) | `30` |
| `--page int` | Page number (when `--paginate` is off) | `1` |
| `--per-page int` | Items per page (max 100) | `30` |
| `--agent-help` | Long, structured help for agents (selected commands) | `false` |
| `--no-extensions` | Skip loading external extensions | `false` |
| `--verbose` | Print HTTP request/response details | `false` |
| `--no-color` | Disable ANSI colors | `false` |
| `--timeout int` | HTTP timeout in seconds | `30` |

**Environment Variables** (13 total; run `gitee help environment` for the full table):

| Variable | Description |
|----------|-------------|
| `GITEE_TOKEN` | Access Token (same as `--token`) |
| `GITEE_HOSTNAME` | Gitee instance address |
| `GITEE_API_URL` | API base URL |
| `GITEE_OUTPUT` | Default output format `table` \| `json` |
| `GITEE_VERBOSE` | Set to `1`/`true`/`yes`/`on` to print HTTP details to stderr |
| `GITEE_TIMEOUT` | HTTP timeout in seconds |
| `GITEE_LIMIT` | Default `--limit` value |
| `GITEE_PAGINATE` | Set to `1`/`true`/`yes`/`on` to enable `--paginate` by default |
| `GITEE_NO_EXTENSIONS` | Set to `1`/`true`/`yes`/`on` to skip extension loading |
| `NO_COLOR` | Non-empty disables colors (standard variable) |
| `GITEE_ALIASES_FILE` | Override aliases file (default `~/.gitee-cli/aliases.json`) |
| `GITEE_EXT_DIR` | Override extension directory (default `~/.gitee-cli/extensions`) |
| `BROWSER` | Browser binary used by `repo browse` |

> Proxy support comes from Go's standard `net/http` variables (`HTTPS_PROXY` / `NO_PROXY`);
> the CLI does not parse them itself.

---

## Shell Completion

```bash
# Bash
echo 'eval "$(gitee completion bash)"' >> ~/.bashrc

# Zsh
gitee completion zsh > "${fpath[1]}/_gitee"

# Fish
gitee completion fish > ~/.config/fish/completions/gitee.fish

# PowerShell
gitee completion powershell >> $PROFILE
```

---

## Enterprise / Private Deployment

```bash
# Login to enterprise instance
gitee auth login --hostname your-company.gitee.com

# Or via environment variable
export GITEE_HOSTNAME=your-company.gitee.com
gitee repo list
```

---

## Changelog

### v0.13.0 (2026-08-19) — Runtime Context Refactor + Architecture Convergence

Core changes: Eliminated `cmd.Global*` global variables, injected `cmd.Runtime` explicit context; unified command organization pattern, eliminated layer inversion.

**🧱 Architecture Refactor**
- Introduced `cmd.Runtime` runtime context struct with `SetContext`/`RuntimeFrom`/`RuntimeFromCmd` injecting config into Cobra command context
- Migrated 110 command files: `NewAPIClient()` → `NewAPIClientFrom(rt)`, `NewCmdkitOptions(client)` → `NewCmdkitOptionsFrom(rt, client)`, `NewDomainClient()` → `NewDomainClientFrom(rt)`
- Eliminated `internal/handler` layer inversion, business logic moved to `internal/domain`
- Abstracted list command execution skeleton, migrated all API list commands eliminating 5-stage boilerplate
- Consolidated repo parsing entry point, deleted redundant `ResolveRepoFromFlag`

**🐛 Fixes**
- `issue comment` false 401 auth error for locked issues + JSON body rejected by Gitee

**📖 Documentation**
- 11-dimension schema completed for 129 agent-help docs + contract test guard
- Agent-help matrix hygiene + philosophy numbers updated
- completion/version n/a signal + contract test traversing command tree

---

## Development

See [`docs/en/developer-guide.md`](docs/en/developer-guide.md)

```bash
# Clone
git clone https://gitee.com/andershsueh/gitee-cli.git && cd gitee-cli

# Build and install
go install ./cmd/gitee

# Test
go test ./... -race -count=1

# Lint
golangci-lint run ./...

# Functional test (requires a built binary + auth)
./test/run.sh
```

Exact test/command counts (packages, test files, test functions, agent-help coverage)
are measured from the current tree — see the Chinese README for the published numbers.

### Project Documentation

| Document | Description |
|----------|-------------|
| [`docs/en/FSD.md`](docs/en/FSD.md) | Functional Specification |
| [`docs/en/SDD.md`](docs/en/SDD.md) | System Design Document |
| [`docs/en/developer-guide.md`](docs/en/developer-guide.md) | Developer Guide |

---

## References

- [Gitee API v5 Documentation](https://gitee.com/api/v5/swagger)
- [Gitee personal access tokens](https://gitee.com/profile/personal_access_tokens)
- [GitHub gh CLI](https://github.com/cli/cli) (design reference)
- [GitLab glab CLI](https://github.com/gitlab-org/cli) (design reference)

---

## License

[Mulan PSL v2](LICENSE)

---

## Buy Me Coffee

If this project helps you, you can buy me a coffee:

![Buy Me Coffee](./etc/BuyMeCoffe.jpg)