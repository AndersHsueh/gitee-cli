# gitee-cli Developer Guide

> ⚠️ **Status notice (v0.13.2, 2026-10 maintenance round)**: This guide is an early design draft;
> some examples are **no longer consistent with the code** (module path, `internal/printer`,
> Viper/oauth2, Makefile/goreleaser, ...). The facts are the code, `gitee --help`,
> `gitee <cmd> --agent-help` and `bash scripts/ci.sh`. The most misleading sections are corrected
> inline.


Version: 1.0.0 (design draft; see banner) | Audience: Contributors / Team Developers

---

## 1. Environment Setup

### 1.1 Required Tools

```bash
# Go 1.27+ (required; go.mod says `go 1.27.3`)
go version    # needs >= 1.27

# golangci-lint (code quality check; CI uses v2.15.0, v2 config schema)
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

# govulncheck (vulnerability scanning; version pinned as a go.mod tool dependency)
go install golang.org/x/vuln/cmd/govulncheck@latest

# Release and CI are in-repo scripts; no Makefile / goreleaser needed
bash scripts/ci.sh --fast   # local quality gate
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
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go mod download
go install ./cmd/gitee      # installs to $GOPATH/bin; use this for daily work
gitee version
```

---

## 2. Repository Bootstrap (current state)

This repository **already exists**; you do not need `go mod init` / `cobra-cli`. Key facts:

- Module path is `gitee-cli` (not `gitee.com/<org>/gitee-cli`), so internal imports read
  `gitee-cli/internal/api`, `gitee-cli/cmd`.
- Dependencies are only Cobra, resty v2, fatih/color, testify (+ indirect);
  **no Viper, no golang.org/x/oauth2**.
- Config is hand-written JSON (`internal/config`), not Viper; auth is PAT, not OAuth.

```bash
# Fetch dependencies + toolchain (per go.mod)
go mod download
go build ./...              # compile self-check
bash scripts/ci.sh --fast   # gofmt + vet + build + test (fast local gate)
```

---

## 3. Development Workflow

### 3.1 Adding a New Subcommand

Example: Adding `gitee issue list`

**Step 1: Create Command File**

See the real `cmd/issue/list.go` (simplified here):

```go
// cmd/issue/list.go
package issue

import (
    "net/url"

    "gitee-cli/cmd"
    "gitee-cli/internal/api"
    "gitee-cli/internal/cmdkit"
    "gitee-cli/internal/utils"
    "gitee-cli/pkg/types"

    "github.com/spf13/cobra"
)

var listFlags struct {
    state    string
    labels   string
    assignee string
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List issues in a repository",
    Example: `  gitee issue list --repo owner/repo
  gitee issue list --repo owner/repo --state closed --labels bug`,
    PreRunE: func(c *cobra.Command, args []string) error {
        if cmd.RuntimeFromCmd(c).AgentHelp {
            cmd.PrintAgentHelp(issueListAgentHelp)   // optional: this command's agent-help text
        }
        return nil
    },
    RunE: runList,
}

func init() {
    f := listCmd.Flags()
    f.StringVar(&listFlags.state, "state", "open", "Filter by state: open|closed|all")
    f.StringVar(&listFlags.labels, "labels", "", "Filter by labels (comma-separated)")
    f.StringVar(&listFlags.assignee, "assignee", "", "Filter by assignee username")
    // --limit / --per-page / --paginate are root persistent flags; do not re-register them.
    IssueCmd.AddCommand(listCmd)
}

func runList(cobraCmd *cobra.Command, args []string) error {
    rt := cmd.RuntimeFromCmd(cobraCmd)
    client, err := cmd.NewAPIClientFrom(rt)
    if err != nil {
        return err
    }

    // List commands go through the cmdkit.RunList skeleton: pagination, table, JSON,
    // empty-result text handled once.
    opts := cmd.NewCmdkitOptionsFrom(rt, client)
    return cmdkit.RunList[types.Issue](opts, cmdkit.ListSpec[types.Issue]{
        ResolveOwnerRepo: func() (string, string, error) {
            return utils.ResolveRepo(rt.Repo, "", nil)
        },
        Path: func(owner, repoName string) string {
            qv := url.Values{}
            qv.Set("state", listFlags.state)
            if listFlags.labels != "" {
                qv.Set("labels", listFlags.labels)
            }
            if listFlags.assignee != "" {
                qv.Set("assignee", listFlags.assignee)
            }
            return api.IssuesList(owner, repoName) + "?" + qv.Encode()
        },
        Columns: []string{"#", "TITLE", "STATE", "LABELS", "UPDATED"},
        Row: func(i types.Issue) []string {
            return []string{i.Number, utils.Truncate(i.Title, 60), i.State, "-", i.UpdatedAt.Format("2006-01-02")}
        },
        EmptyText: func(o, r string) string { return "No issues found in " + o + "/" + r },
        ErrCtx:    "listing issues",
    })
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

Build endpoint paths only via `internal/api/endpoints.go` builders; **never `fmt.Sprintf` URLs in cmd/**:

```go
// internal/api/endpoints.go
func IssuesList(owner, repo string) string {
    return "/repos/" + owner + "/" + repo + "/issues"
}

func IssueGet(owner, repo string, number int) string {
    return IssuesList(owner, repo) + "/" + strconv.Itoa(number)
}
```

**Step 4: Add/Update Type Definitions**

```go
// pkg/types/issue.go — defined in SDD, extend fields as needed
```

**Step 5: Write Tests (see Section 4)**

---

### 3.2 Adding New API Endpoint Calls

Reuse the generated Client methods (`Get/Post/Patch/PatchForm/Put/Delete/DeleteWithBody/UploadFile`)
instead of adding business methods to `internal/api`:

```go
// Read: pass a result pointer
var issue types.Issue
err := client.Get(ctx, api.IssueGet(owner, repo, number), &issue)

// Write: JSON body
err := client.Post(ctx, api.IssuesCreate(owner, repo), req, &created)

// Write: some Gitee v5 write paths only accept formData (issue close/reopen/edit)
err := client.PatchForm(ctx, api.IssuesEdit(owner, number), url.Values{
    "repo":  {repo},
    "state": {"closed"},
})

// Lists: prefer FetchListOrAll / cmdkit.RunList, never hand-roll paginate branches
issues, err := api.FetchListOrAll[types.Issue](ctx, client, path, paginate, perPage, limit, page, "listing issues")
```

---

### 3.3 Repository Context Detection

Resolve repositories through `internal/utils`; do not parse git remotes inside commands:

```go
// utils.ParseOwnerRepo("owner/repo") → owner, repo
// utils.ResolveRepo(owner, repo, args) → resolves --repo / positional arg / git remote in order
owner, repoName, err := utils.ResolveRepo(rt.Repo, "", args)
```

- `--repo` / `-R` (some commands) wins when passed;
- then positional args (e.g. `gitee repo view owner/repo`);
- finally the current directory's git remote (`git remote get-url origin`).

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

    client := api.NewClient("test-token", ts.URL, false, 30)  // token, baseURL, verbose, timeout

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

| Module | Suggested coverage |
|--------|--------------------|
| `internal/api/` | High (HTTP client and error mapping) |
| `internal/config/` | High (permissions, atomic write, precedence) |
| `internal/domain/` | Medium (currently a weak spot; new write paths should ship httptest) |
| `pkg/types/` | Not enforced |
| `cmd/` | Medium (agent-help contract tests cover the text) |

> Current state: roughly 30% repo-wide (historically low); CI has no coverage gate yet —
> new features are expected to bring their own httptest cases.

---

## 5. Code Standards

### 5.1 Lint Configuration (`.golangci.yml`)

The repo's `.golangci.yml` uses the **v2 schema** (`version: "2"`) with the default linter set
(errcheck / govet / ineffassign / staticcheck / unused) plus targeted exemptions for unchecked
returns in test files. gofmt is enforced separately by `scripts/ci.sh`.

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

### 6.4 Use a Token Directly

```bash
# Recommended: environment variable (not written to disk, not in shell history)
export GITEE_TOKEN=your_access_token
gitee auth status

# Or log in with an explicit token (written to ~/.gitee-cli/settings.json)
gitee auth login --token your_access_token
```

---

## 7. Build and Release

### 7.1 Local Build

```bash
go install ./cmd/gitee           # installs to $GOPATH/bin (daily development)
go build -o gitee ./cmd/gitee    # one-off local build (not on PATH)
```

### 7.2 Cross-Compilation

```bash
# Manual cross-build (release script does the same)
GOOS=linux   GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee
GOOS=linux   GOARCH=arm64 go build -o dist/gitee-linux-arm64 ./cmd/gitee
GOOS=darwin  GOARCH=amd64 go build -o dist/gitee-darwin-amd64 ./cmd/gitee
GOOS=darwin  GOARCH=arm64 go build -o dist/gitee-darwin-arm64 ./cmd/gitee
GOOS=windows GOARCH=amd64 go build -o dist/gitee-windows-amd64.exe ./cmd/gitee
```

### 7.3 Release

```bash
# Tag and push first
git tag -a v0.13.2 -m v0.13.2 && git push origin v0.13.2

# Build 5 platform artifacts + SHA256SUMS + create Gitee release + upload assets (idempotent)
GITEE_TOKEN=... bash scripts/release.sh --publish
```

> No Makefile, no goreleaser. Releases use `scripts/release.sh` (build/package/checksums) and
> `scripts/publish-release.sh` (create release + upload assets via `gitee release create/upload`).

---

## 8. CI/CD Pipeline

There are **two CI backends but one gate definition**:

- `.github/workflows/ci.yml` (GitHub Actions mirror)
- `.workflow/ci.yml` (Gitee Go)
- both call `scripts/ci.sh`: gofmt → go vet → golangci-lint (optional) → build →
  `go test -race -count=1` → cross-compile → govulncheck

Reproduce CI locally:

```bash
bash scripts/ci.sh          # full (cross-compile + govulncheck)
bash scripts/ci.sh --fast   # skip cross-compile + govulncheck for a quick loop
```

---

## 9. Project Milestones

| Phase | Target Commands | Description |
|-------|----------------|-------------|
| **M0** | `gitee auth` full suite | PAT auth (no OAuth2 device flow), token storage, `auth status/logout/token` |
| **M1** | `gitee repo` full suite + `gitee api` | Repo CRUD, raw API access, config module |
| **M2** | `gitee issue` full suite | Issue CRUD, comments, label management |
| **M3** | `gitee pr` full suite | PR CRUD, Merge, Review, diff/checks |
| **M4** | `gitee gist` + `gitee search` | Gist management, three search types |
| **M5** | `gitee org` + shell completion | Org management, Bash/Zsh/Fish/PowerShell completion (i18n not implemented) |
| **M6** | Enterprise deployment + multi-account + plugin system | `--hostname` extension, credentials multi-host, external command discovery |

---

## 10. References

| Resource | Link |
|----------|------|
| Gitee API v5 Swagger | https://gitee.com/api/v5/swagger |
| Gitee personal access tokens | https://gitee.com/profile/personal_access_tokens |
| Cobra Official Documentation | https://pkg.go.dev/github.com/spf13/cobra |
| Resty HTTP Client | https://github.com/go-resty/resty |
| glab Project (best reference) | https://github.com/gitlab-org/cli |
| gh CLI Project | https://github.com/cli/cli |
| Go Project Layout Standard | https://github.com/golang-standards/project-layout |
| Conventional Commits | https://www.conventionalcommits.org |
| golangci-lint Rules Documentation | https://golangci-lint.run/usage/linters/ |