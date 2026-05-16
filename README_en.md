# gitee-cli

> Gitee Official CLI Tool — Complete Gitee operations in the terminal, equivalent to GitHub's `gh`

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.3.0-orange.svg)]()

---

## Features

```
gitee auth        # Authentication (login / logout / status / token)               ✅ Implemented
gitee issue       # Issue Management (list / create / view / edit / close / reopen / comment) ✅ Implemented
gitee config      # Configuration (set / get / list / unset)                        ✅ Implemented
gitee search      # Search repos / issues / users                                   ✅ Implemented
gitee completion  # Shell Completion (bash / zsh / fish / powershell)                ✅ Implemented
gitee version     # Version Info                                                    ✅ Implemented
gitee repo        # Repo Management (list / create / view / fork / edit / delete / clone / browse / branches) ✅ Implemented
gitee pr          # Pull Request (list / create / view / merge / review / comments / diff) ✅ Implemented
gitee gist        # Gist (list / create / view / edit / delete / star / unstar)     ✅ Implemented
gitee org         # Organization (list / view / members / repos / issues)           ✅ Implemented
gitee api         # Direct Gitee API v5 calls (GET / POST / PUT / PATCH / DELETE)   ✅ Implemented
```

---

## Quick Start

### Build from Source

```bash
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go build -o gitee ./cmd/gitee
sudo mv gitee /usr/local/bin/
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

**Homebrew (macOS / Linux)**

```bash
brew install gitee/tap/gitee
```

**Binary Download**

Download binaries from [Releases](https://gitee.com/gitee/gitee-cli/releases):

| Platform | Filename |
|----------|----------|
| Linux amd64 | `gitee-linux-amd64` |
| Linux arm64 | `gitee-linux-arm64` |
| macOS amd64 | `gitee-darwin-amd64` |
| macOS arm64 (M1/M2) | `gitee-darwin-arm64` |
| Windows amd64 | `gitee-windows-amd64.exe` |

**Build from Source**

```bash
git clone https://gitee.com/gitee/gitee-cli.git
cd gitee-cli
go build -o gitee ./cmd/gitee
sudo mv gitee /usr/local/bin/
```

### Authentication

```bash
gitee auth login
# Follow the prompts to complete authorization in browser, token auto-saved to ~/.gitee-cli/settings.json
```

Or use existing token:

```bash
echo "your_access_token" | gitee auth login --with-token
```

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
gitee pr merge 18 --squash --delete-branch

# Review PR
gitee pr review 18 --approve
gitee pr review 18 --request-changes --body "Please add unit tests"
```

### Search

```bash
# Search Go repos, sort by stars
gitee search repos "web framework" --language go --sort stars

# Search users
gitee search users "john"

# Search issues
gitee search issues "memory leak" --state open
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

# View response headers
gitee api GET /user --include
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
| `--token string` | Access Token (overrides config) | from config |
| `--hostname string` | Gitee instance address (enterprise) | `gitee.com` |
| `-o, --output string` | Output format: `table` \| `json` \| `csv` | `table` |
| `-P, --paginate` | Auto-paginate, fetch all results | `false` |
| `--limit int` | Max results | `0` (unlimited) |
| `--page int` | Page number | `1` |
| `--per-page int` | Items per page (max 100) | `30` |
| `--verbose` | Print HTTP request/response details | `false` |
| `--no-color` | Disable ANSI colors | `false` |
| `--no-interaction` | Silent mode (CI/CD) | `false` |
| `--lang string` | UI language: `zh` \| `en` | `zh` |
| `--timeout int` | Network timeout in seconds | `30` |

**Environment Variables:**

| Variable | Description |
|----------|-------------|
| `GITEE_TOKEN` | Access Token |
| `GITEE_HOSTNAME` | Gitee instance address |
| `GITEE_OUTPUT` | Default output format |
| `GITEE_API_URL` | API root URL |
| `HTTPS_PROXY` | HTTPS proxy |
| `NO_COLOR` | Disable colors (standard) |

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

## Development

See [`documents/developer-guide.md`](documents/developer-guide.md)

```bash
# Clone
git clone https://gitee.com/gitee/gitee-cli.git && cd gitee-cli

# Build
go build -o bin/gitee ./cmd/gitee

# Test
go test ./... -race -count=1

# Lint
golangci-lint run ./...

# Functional test
./test/run.sh
```

### Project Documentation

| Document | Description |
|----------|-------------|
| [`documents/FSD.md`](documents/FSD.md) | Functional Specification |
| [`documents/SDD.md`](documents/SDD.md) | System Design Document |
| [`documents/developer-guide.md`](documents/developer-guide.md) | Developer Guide |

---

## References

- [Gitee API v5 Documentation](https://gitee.com/api/v5/swagger)
- [Gitee OAuth2 Documentation](https://gitee.com/api/v5/oauth_doc)
- [GitHub gh CLI](https://github.com/cli/cli) (design reference)
- [GitLab glab CLI](https://github.com/gitlab-org/cli) (design reference)

---

## License

[Apache 2.0](LICENSE)

---

## Buy Me Coffee

If this project helps you, you can buy me a coffee:

![Buy Me Coffee](./etc/BuyMeCoffe.jpg)