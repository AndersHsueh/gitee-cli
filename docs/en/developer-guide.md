# gitee-cli Developer Guide

Version: 1.0.0 | Audience: Contributors / Team Developers

---

## 1. Environment Setup

### 1.1 Required Tools

```bash
# Go 1.23+ (required)
go version    # needs >= 1.23

# Cobra CLI (command template generator)
go install github.com/spf13/cobra-cli@latest

# golangci-lint (code quality check)
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin v1.59.0

# goreleaser (release automation)
go install github.com/goreleaser/goreleaser@latest

# govulncheck (vulnerability scanning)
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### 1.2 Recommended Tools

```bash
# goimports (import organization)
go install golang.org/x/tools/cmd/goimports@latest

# mockgen (mock generation)
go install go.uber.org/mock/mockgen@latest
```

### 1.3 Clone and Initialize

```bash
git clone https://gitee.com/<your-user>/gitee-cli.git
cd gitee-cli
go mod download
go build -o bin/gitee ./cmd/gitee
./bin/gitee --version
```

---

## 2. Project Initialization (New Projects)

```bash
# Initialize Go module (module path follows gitee.com/<org>/gitee-cli)
go mod init gitee.com/gitee/gitee-cli

# Add dependencies
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/go-resty/resty/v2
go get golang.org/x/oauth2
go get github.com/fatih/color
go get github.com/stretchr/testify

# Use Cobra CLI to generate skeleton
cobra-cli init --viper --author "Gitee Team" --license apache
cobra-cli add auth
cobra-cli add repo
cobra-cli add issue
cobra-cli add pr
cobra-cli add gist
cobra-cli add org
cobra-cli add search
cobra-cli add api
cobra-cli add config
cobra-cli add completion
```

---

## 3. Development Workflow

### 3.1 Adding a New Subcommand

Example: Adding `gitee issue list`

**Step 1: Create Command File**

```go
// cmd/issue/list.go
package issue

import (
    "context"
    "fmt"

    "github.com/spf13/cobra"
    "gitee.com/gitee/gitee-cli/internal/api"
    "gitee.com/gitee/gitee-cli/internal/printer"
    "gitee.com/gitee/gitee-cli/pkg/types"
)

var listFlags struct {
    State    string
    Labels   string
    Assignee string
    Repo     string
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List Issues",
    Long:  "List issues in current or specified repository, supports filtering by state, labels, assignee.",
    RunE:  runList,
}

func init() {
    f := listCmd.Flags()
    f.StringVarP(&listFlags.State, "state", "s", "open", "Issue state: open/closed/all")
    f.StringVarP(&listFlags.Labels, "labels", "l", "", "Label filter (comma-separated)")
    f.StringVarP(&listFlags.Assignee, "assignee", "a", "", "Assignee filter")
    f.StringVarP(&listFlags.Repo, "repo", "R", "", "Repository (owner/name format)")

    IssueCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    client, err := api.NewClientFromConfig()
    if err != nil {
        return err
    }

    owner, repo, err := resolveRepo(listFlags.Repo)
    if err != nil {
        return err
    }

    var issues []types.Issue
    path := fmt.Sprintf("/repos/%s/%s/issues?state=%s", owner, repo, listFlags.State)
    if listFlags.Labels != "" {
        path += "&labels=" + listFlags.Labels
    }
    if err := client.List(ctx, path, nil, &issues); err != nil {
        return err
    }

    outputFmt, _ := cmd.Flags().GetString("output")
    p := printer.New(outputFmt, cmd.OutOrStdout())
    p.PrintHeader("#", "TITLE", "STATE", "LABELS", "ASSIGNEE", "UPDATED")
    for _, i := range issues {
        p.PrintRow(
            fmt.Sprintf("#%d", i.Number),
            truncate(i.Title, 60),
            i.State,
            joinLabels(i.Labels, 3),
            firstAssignee(i.Assignees),
            humanTime(i.UpdatedAt),
        )
    }
    return p.Flush()
}
```

**Step 2: Register with Parent Command**

```go
// cmd/issue/issue.go
var IssueCmd = &cobra.Command{
    Use:   "issue",
    Short: "Manage Issues",
}

// cmd/root.go
func init() {
    rootCmd.AddCommand(issue.IssueCmd)
}
```

**Step 3: Add API Endpoint (if needed)**

```go
// internal/api/endpoints.go
const (
    EndpointIssues = "/repos/%s/%s/issues"
    EndpointIssue  = "/repos/%s/%s/issues/%d"
    EndpointIssueComments = "/repos/%s/%s/issues/%d/comments"
)
```

**Step 4: Add/Update Type Definitions**

```go
// pkg/types/issue.go — defined in SDD, extend fields as needed
```

**Step 5: Write Tests (see Section 4)**

---

### 3.2 Adding New API Endpoint Calls

```go
// Add semantic methods in internal/api/client.go
func (c *client) ListIssues(ctx context.Context, owner, repo, state string) ([]types.Issue, error) {
    var issues []types.Issue
    path := fmt.Sprintf("/repos/%s/%s/issues?state=%s", owner, repo, state)
    err := c.List(ctx, path, &PageOptions{PerPage: 30}, &issues)
    return issues, err
}

func (c *client) CreateIssue(ctx context.Context, owner, repo string, body types.IssueCreateRequest) (*types.Issue, error) {
    var issue types.Issue
    path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
    err := c.Post(ctx, path, body, &issue)
    return &issue, err
}
```

---

### 3.3 Repository Context Detection

Many commands need to auto-detect current repository (similar to `gh`), parsed from git remote:

```go
// internal/git/remote.go
func DetectRepo() (owner, repo string, err error) {
    // 1. Check --repo flag
    // 2. Parse git remote origin URL
    //    - https://gitee.com/owner/repo.git
    //    - git@gitee.com:owner/repo.git
    // 3. Return owner, repo
    remoteURL, err := exec.Command("git", "remote", "get-url", "origin").Output()
    // ... parsing logic
}
```

---

## 4. Testing Standards

### 4.1 Unit Tests

Each `*.go` file has a corresponding `*_test.go`, using `httptest` to mock HTTP:

```go
// internal/api/client_test.go
func TestClient_ListIssues(t *testing.T) {
    // Set up mock server
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "GET", r.Method)
        assert.Contains(t, r.URL.Path, "/issues")
        assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("X-Total-Count", "2")
        json.NewEncoder(w).Encode([]types.Issue{
            {Number: 1, Title: "First Issue", State: "open"},
            {Number: 2, Title: "Second Issue", State: "closed"},
        })
    }))
    defer ts.Close()

    client := api.NewClient(
        api.WithBaseURL(ts.URL),
        api.WithToken("test-token"),
    )

    issues, err := client.ListIssues(context.Background(), "owner", "repo", "all")
    require.NoError(t, err)
    require.Len(t, issues, 2)
    assert.Equal(t, 1, issues[0].Number)
    assert.Equal(t, "First Issue", issues[0].Title)
}
```

### 4.2 Command Layer Tests (Output Verification)

```go
// cmd/issue/list_test.go
func TestIssueListCommand(t *testing.T) {
    // Set up mock API
    ts := setupMockServer(t, mockIssuesResponse)
    defer ts.Close()

    // Capture stdout
    buf := &bytes.Buffer{}
    cmd := newTestRootCmd(ts.URL)
    cmd.SetOut(buf)
    cmd.SetArgs([]string{"issue", "list", "--output", "json"})

    err := cmd.Execute()
    require.NoError(t, err)

    var result []map[string]interface{}
    require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
    assert.Len(t, result, 2)
}
```

### 4.3 Running Tests

```bash
go test ./...                    # Run all tests
go test ./cmd/... -v             # Command layer verbose
go test ./internal/api/... -v   # API layer verbose
go test ./... -cover             # Coverage
go test ./... -race              # Race detection (CI required)
go test ./... -count=1           # Disable cache (CI use)
```

### 4.4 Coverage Requirements

| Module | Min Coverage |
|--------|-------------|
| `internal/api/` | 80% |
| `internal/auth/` | 70% |
| `internal/printer/` | 90% |
| `pkg/types/` | Not enforced |
| `cmd/` | 60% |

---

## 5. Code Standards

### 5.1 Lint Configuration (`.golangci.yml`)

```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - gosec
    - revive
    - exhaustive

linters-settings:
  goimports:
    local-prefixes: gitee.com/gitee/gitee-cli
```

Run:

```bash
golangci-lint run ./...
golangci-lint run --fix ./...   # Auto-fix
```

### 5.2 Formatting

```bash
go fmt ./...        # Format code
goimports -w .      # Organize imports (group: stdlib / external / internal)
```

### 5.3 Commit Standards (Conventional Commits)

Format: `<type>(<scope>): <description>`

| Type | Use Case |
|------|----------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation change |
| `test` | Add/modify tests |
| `refactor` | Refactoring (non-functional change) |
| `chore` | Dependency update, build scripts |
| `perf` | Performance optimization |
| `ci` | CI/CD configuration change |

Examples:

```
feat(repo): add repo fork command
fix(auth): handle expired token gracefully
docs(readme): add installation guide for macOS
test(api): add client pagination test coverage
refactor(printer): extract interface for output formats
chore(deps): bump resty to v2.12.0
```

### 5.4 Code Review Checklist

```
Before submitting:
[ ] go fmt && goimports run
[ ] golangci-lint with no warnings
[ ] All API calls have error handling (don't ignore error)
[ ] Network requests pass context.Context
[ ] Token not in logs, stdout, or any visible output
[ ] New feature has corresponding unit tests
[ ] Tests cover happy path + error path
[ ] --help output description is accurate
[ ] New flag has default value documentation
```

---

## 6. Debugging Tips

### 6.1 Enable Verbose Logging

```bash
gitee --verbose repo list
# Output:
# → GET https://gitee.com/api/v5/user/repos?page=1&per_page=30
# ← HTTP 200 (245ms) [3 items]
```

### 6.2 Use `gitee api` Command to Test Endpoints Directly

```bash
# Get current user info
gitee api GET /user

# List repos (with pagination)
gitee api GET /user/repos?type=owner&per_page=5

# Create test repo
gitee api POST /user/repos --field name=test-cli-repo --field private=true --field auto_init=true

# Delete test repo
gitee api DELETE /repos/owner/test-cli-repo
```

### 6.3 Override API Address (Test Private Deployment)

```bash
gitee --api-url "https://your-enterprise.gitee.com/api/v5" repo list
# or
GITEE_API_URL="https://your-enterprise.gitee.com/api/v5" gitee repo list
```

### 6.4 Use Token Directly (Skip OAuth)

```bash
gitee auth login --with-token
# Read token from stdin:
echo "your_access_token" | gitee auth login --with-token
```

---

## 7. Build and Release

### 7.1 Local Build

```bash
make build          # Compile to bin/gitee
make install        # Install to $GOPATH/bin
```

### 7.2 Cross-Compilation

```bash
make cross-build
# Output:
# dist/gitee-linux-amd64
# dist/gitee-linux-arm64
# dist/gitee-darwin-amd64
# dist/gitee-darwin-arm64
# dist/gitee-windows-amd64.exe
```

### 7.3 Release

```bash
# Tag first
git tag v0.1.0
git push origin v0.1.0

# goreleaser auto-build + upload
make release
```

---

## 8. CI/CD Pipeline

### 8.1 GitHub Actions / Gitee Go CI Example

```yaml
# .gitee/workflows/ci.yml
name: CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go mod download
      - run: go vet ./...
      - run: golangci-lint run ./...
      - run: go test ./... -v -race -count=1
      - run: govulncheck ./...

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: make cross-build
```

---

## 9. Project Milestones

| Phase | Target Commands | Description |
|-------|----------------|-------------|
| **M0** | `gitee auth` full suite | OAuth2 Device Flow, Token storage, `auth status/logout/token` |
| **M1** | `gitee repo` full suite + `gitee api` | Repo CRUD, raw API access, config module |
| **M2** | `gitee issue` full suite | Issue CRUD, comments, label management |
| **M3** | `gitee pr` full suite | PR CRUD, Merge, Review, diff/checks |
| **M4** | `gitee gist` + `gitee search` | Gist management, three search types |
| **M5** | `gitee org` + Shell completion + i18n | Org management, Bash/Zsh/Fish completion, zh/en switch |
| **M6** | Enterprise deployment + multi-account + plugin system | `--hostname` extension, credentials multi-host, external command discovery |

---

## 10. References

| Resource | Link |
|----------|------|
| Gitee API v5 Swagger | https://gitee.com/api/v5/swagger |
| Gitee OAuth2 Documentation | https://gitee.com/api/v5/oauth_doc |
| Cobra Official Documentation | https://pkg.go.dev/github.com/spf13/cobra |
| Viper Official Documentation | https://pkg.go.dev/github.com/spf13/viper |
| Resty HTTP Client | https://github.com/go-resty/resty |
| glab Project (best reference) | https://github.com/gitlab-org/cli |
| gh CLI Project | https://github.com/cli/cli |
| Go Project Layout Standard | https://github.com/golang-standards/project-layout |
| Conventional Commits | https://www.conventionalcommits.org |
| golangci-lint Rules Documentation | https://golangci-lint.run/usage/linters/ |