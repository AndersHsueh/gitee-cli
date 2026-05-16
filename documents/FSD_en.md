# gitee-cli Functional Specification Document

Version: 1.0.0 | Language: Go 1.23+ | Reference: Gitee API v5 + GitHub `gh` CLI

---

## 1. Project Overview

### 1.1 Project Information

| Field | Value |
|-------|-------|
| Project Name | `gitee-cli` |
| Type | Command Line Tool (CLI) |
| Language | Go 1.23+ |
| Framework | Cobra + Viper |
| Goal | Become Gitee's official CLI tool, equivalent to GitHub `gh` |
| API Base | `https://gitee.com/api/v5` |
| Reference Projects | [`gh`](https://github.com/cli/cli), [`glab`](https://github.com/cli/go-gitlab) |

### 1.2 Core Features Summary

```
gitee auth login|logout|status|refresh|token   # OAuth2 authentication
gitee repo list|create|clone|fork|view|edit|delete|browse  # Repository management
gitee issue list|create|view|edit|close|reopen|comment     # Issue management
gitee pr list|create|view|merge|close|reopen|review|diff|checks  # Pull Request
gitee gist list|create|view|edit|delete|star|clone         # Gist
gitee org list|view|member|repo|issue                      # Organization
gitee search repos|issues|users                            # Search
gitee api <method> <path> [--field key=val]                # Raw API access
gitee config set|get|list|unset                            # Configuration
gitee completion bash|zsh|fish|powershell                  # Shell completion
```

---

## 2. Global Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--hostname string` | string | Gitee server address (enterprise/private deployment) | `gitee.com` |
| `--token string` | string | OAuth Access Token (priority over config) | from config |
| `-o, --output string` | string | Output format (`table`\|`json`\|`csv`) | `table` |
| `-P, --paginate` | bool | Auto-paginate through all results | `false` |
| `--page int` | int | Page number | `1` |
| `--per-page int` | int | Items per page (max 100) | `30` |
| `--limit int` | int | Max results (0 = unlimited) | `0` |
| `-c, --config string` | string | Config file path | `~/.gitee-cli/settings.json` |
| `--api-url string` | string | API root URL | `https://gitee.com/api/v5` |
| `--timeout int` | int | Network timeout in seconds | `30` |
| `-v, --verbose` | bool | Output debug info (HTTP request/response details) | `false` |
| `--no-color` | bool | Disable ANSI color output | `false` |
| `--no-interaction` | bool | Silent mode, disable interactive prompts (CI/CD) | `false` |
| `--insecure` | bool | Allow insecure HTTPS connections | `false` |
| `--lang string` | string | UI language (`zh`\|`en`) | `zh` |

**Environment Variable Mapping (priority below flag, above config):**

| Environment Variable | Corresponding Flag |
|---------------------|-------------------|
| `GITEE_TOKEN` | `--token` |
| `GITEE_HOSTNAME` | `--hostname` |
| `GITEE_OUTPUT` | `--output` |
| `GITEE_API_URL` | `--api-url` |
| `HTTPS_PROXY` / `HTTP_PROXY` | Proxy settings |
| `NO_COLOR` | `--no-color` |

---

## 3. Command Detailed Specifications

### 3.1 `gitee auth` — Authentication Management

**Corresponding Gitee API:** OAuth2 Device Flow (`/oauth/authorizations/device/*`)

| Subcommand | Gitee API Endpoint | Description |
|------------|-------------------|-------------|
| `gitee auth login` | `POST /oauth/authorizations/device/code` + polling | Start OAuth2 device flow, get Access Token |
| `gitee auth logout` | Local operation | Delete Token from `~/.gitee-cli/settings.json` |
| `gitee auth status` | `GET /user` | Show current login status (username, token expiry) |
| `gitee auth refresh` | `POST /oauth/token?grant_type=refresh_token` | Refresh expired Token |
| `gitee auth token` | Local operation | Output current Access Token (for piping) |

**`gitee auth login` Full Flow:**
```
1. POST /oauth/authorizations/device/code
   Body: { client_id, scope }
   → device_code, user_code, verification_uri, expires_in, interval

2. Display prompt:
   ! First copy your one-time code: XXXX-XXXX
   - Press Enter to open gitee.com/device in your browser

3. Open browser / or in --no-interaction mode print URL

4. Poll POST /oauth/authorizations/device/token
   Body: { client_id, device_code, grant_type="urn:ietf:params:oauth:grant-type:device_code" }
   → access_token, token_type, scope, refresh_token

5. Write access_token to ~/.gitee-cli/settings.json (0600 permissions)

6. GET /user → verify and display "✓ Logged in as <username>"
```

**`gitee auth login` Parameters:**

| Parameter | Description |
|-----------|-------------|
| `--scopes string` | Requested OAuth permission scopes (default: `user_info,projects,pull_requests,issues,notes,gists`) |
| `--with-token` | Read existing Token from stdin and save (skip OAuth flow) |

---

### 3.2 `gitee repo` — Repository Management

**Corresponding Gitee API:** `/repos`, `/user/repos`, `/orgs/{org}/repos`

| Subcommand | Gitee API | Description |
|------------|-----------|-------------|
| `gitee repo list` | `GET /user/repos` | List current user's repos |
| `gitee repo create` | `POST /user/repos` | Create new repo |
| `gitee repo clone <repo>` | call `git clone` | Clone repo to local |
| `gitee repo fork <repo>` | `POST /repos/{owner}/{repo}/forks` | Fork repo |
| `gitee repo view <repo>` | `GET /repos/{owner}/{repo}` | View repo details |
| `gitee repo edit <repo>` | `PATCH /repos/{owner}/{repo}` | Edit repo settings |
| `gitee repo delete <repo>` | `DELETE /repos/{owner}/{repo}` | Delete repo (requires `--yes` confirmation) |
| `gitee repo browse <repo>` | open browser | Open repo page in browser |

**`gitee repo list` Output Fields:**

| Field | API Field | Description |
|-------|-----------|-------------|
| `NAME` | `full_name` | owner/repo format |
| `VISIBILITY` | `private` + `internal` | public/private/internal |
| `LANGUAGE` | `language` | Primary programming language |
| `STARS` | `stargazers_count` | Star count |
| `FORKS` | `forks_count` | Fork count |
| `UPDATED` | `updated_at` | Last update time (relative) |

**`gitee repo list` Parameters:**

| Parameter | Description | Default |
|----------|-------------|---------|
| `--owner string` | Specify user/organization name | current logged-in user |
| `--visibility string` | `public`\|`private`\|`internal`\|`all` | `all` |
| `--sort string` | `created`\|`updated`\|`pushed`\|`full_name` | `updated` |
| `--direction string` | `asc`\|`desc` | `desc` |
| `--type string` | `owner`\|`member`\|`all` | `owner` |

**`gitee repo create` Parameters:**

| Parameter | Required | API Field | Description |
|-----------|----------|-----------|-------------|
| `--name string` | ✅ | `name` | Repo name |
| `--description string` | ❌ | `description` | Repo description |
| `--private` | ❌ | `private=true` | Set as private |
| `--auto-init` | ❌ | `auto_init=true` | Initialize README (default true) |
| `--gitignore string` | ❌ | `gitignore_template` | .gitignore template name |
| `--license string` | ❌ | `license_template` | License template name |
| `--org string` | ❌ | — | Create under organization (use `POST /orgs/{org}/repos`) |

---

### 3.3 `gitee issue` — Issue Management

**Corresponding Gitee API:** `/repos/{owner}/{repo}/issues`

| Subcommand | Gitee API | Description |
|------------|-----------|-------------|
| `gitee issue list` | `GET /repos/{owner}/{repo}/issues` | List issues |
| `gitee issue create` | `POST /repos/{owner}/{repo}/issues` | Create issue |
| `gitee issue view <number>` | `GET /repos/{owner}/{repo}/issues/{number}` | View details |
| `gitee issue edit <number>` | `PATCH /repos/{owner}/{repo}/issues/{number}` | Edit issue |
| `gitee issue close <number>` | `PATCH /repos/{owner}/{repo}/issues/{number}` body `state=closed` | Close issue |
| `gitee issue reopen <number>` | `PATCH /repos/{owner}/{repo}/issues/{number}` body `state=open` | Reopen issue |
| `gitee issue comment <number>` | `POST /repos/{owner}/{repo}/issues/{number}/comments` | Add comment |
| `gitee issue label list` | `GET /repos/{owner}/{repo}/labels` | List repo labels |

**`gitee issue list` Parameters:**

| Parameter | API Field | Default |
|-----------|-----------|---------|
| `--state string` | `state` | `open` |
| `--labels string` | `labels` (comma-separated) | — |
| `--assignee string` | `assignee` | — |
| `--milestone string` | `milestone` | — |
| `--repo string` | `{owner}/{repo}` | detect from git remote |

**`gitee issue list` Output Fields:**

| Field | API Field | Description |
|-------|-----------|-------------|
| `#` | `number` | Issue number |
| `TITLE` | `title` | Title (truncated to 60 chars) |
| `STATE` | `state` | open/closed |
| `LABELS` | `labels[].name` | Labels (show max 3) |
| `ASSIGNEE` | `assignee.login` | Assignee |
| `UPDATED` | `updated_at` | Update date (relative) |

**`gitee issue create` Parameters:**

| Parameter | API Field | Required |
|-----------|-----------|----------|
| `--title string` | `title` | ✅ |
| `--body string` | `body` | ❌ |
| `--labels string` | `labels` (comma-separated) | ❌ |
| `--assignees string` | `assignees` (comma-separated) | ❌ |
| `--milestone int` | `milestone_number` | ❌ |
| `--repo string` | `{owner}/{repo}` | detect from git remote |

---

### 3.4 `gitee pr` — Pull Request Management

**Corresponding Gitee API:** `/repos/{owner}/{repo}/pulls`

| Subcommand | Gitee API | Description |
|------------|-----------|-------------|
| `gitee pr list` | `GET /repos/{owner}/{repo}/pulls` | List PRs |
| `gitee pr create` | `POST /repos/{owner}/{repo}/pulls` | Create PR |
| `gitee pr view <number>` | `GET /repos/{owner}/{repo}/pulls/{number}` | View PR details |
| `gitee pr merge <number>` | `PUT /repos/{owner}/{repo}/pulls/{number}/merge` | Merge PR |
| `gitee pr close <number>` | `PATCH /repos/{owner}/{repo}/pulls/{number}` `state=closed` | Close PR |
| `gitee pr reopen <number>` | `PATCH /repos/{owner}/{repo}/pulls/{number}` `state=open` | Reopen PR |
| `gitee pr review <number>` | `POST /repos/{owner}/{repo}/pulls/{number}/reviews` | Review PR |
| `gitee pr diff <number>` | `GET /repos/{owner}/{repo}/pulls/{number}` Accept: diff | Output diff content |
| `gitee pr checks <number>` | `GET /repos/{owner}/{repo}/statuses/{sha}` | CI check status |

**`gitee pr list` Parameters:**

| Parameter | API Field | Default |
|-----------|-----------|---------|
| `--state string` | `state` | `open` |
| `--author string` | `author` | — |
| `--head string` | `head` | — |
| `--base string` | `base` | — |
| `--sort string` | `sort` | `created` |
| `--direction string` | `direction` | `desc` |

**`gitee pr merge` Parameters:**

| Parameter | API Field | Description |
|-----------|-----------|-------------|
| `--squash` | `merge_method=squash` | Squash merge |
| `--rebase` | `merge_method=rebase` | Rebase merge |
| `--merge-commit-message string` | `merge_commit_message` | Merge commit message |
| `--delete-branch` | — | Delete source branch after merge |

---

### 3.5 `gitee gist` — Gist

**Corresponding Gitee API:** `/gists`

| Subcommand | Gitee API | Description |
|------------|-----------|-------------|
| `gitee gist list` | `GET /gists` | List user's gists |
| `gitee gist create [files...]` | `POST /gists` | Create gist |
| `gitee gist view <id>` | `GET /gists/{id}` | View gist content |
| `gitee gist edit <id>` | `PATCH /gists/{id}` | Edit gist |
| `gitee gist delete <id>` | `DELETE /gists/{id}` | Delete gist |
| `gitee gist star <id>` | `PUT /gists/{id}/star` | Star gist |
| `gitee gist clone <id>` | call `git clone` | Clone gist as local git repo |

**`gitee gist create` Parameters:**

| Parameter | API Field | Description |
|-----------|-----------|-------------|
| `--description string` | `description` | Gist description |
| `--public` | `public=true` | Make public (default private) |
| `--from-stdin` | — | Read content from stdin (with `--filename`) |
| `--filename string` | — | Filename in stdin mode |

**`gitee gist list` Output Fields:**

| Field | API Field | Description |
|-------|-----------|-------------|
| `ID` | `id` | Gist ID |
| `DESCRIPTION` | `description` | Description (truncated to 50 chars) |
| `FILES` | `files` count | Number of files |
| `PUBLIC` | `public` | Whether public |
| `UPDATED` | `updated_at` | Update time |

---

### 3.6 `gitee org` — Organization Management

**Corresponding Gitee API:** `/orgs`, `/orgs/{org}`

| Subcommand | Gitee API | Description |
|------------|-----------|-------------|
| `gitee org list` | `GET /user/orgs` | List user's organizations |
| `gitee org view <org>` | `GET /orgs/{org}` | View org details |
| `gitee org member list <org>` | `GET /orgs/{org}/members` | List org members |
| `gitee org repo list <org>` | `GET /orgs/{org}/repos` | List org repos |
| `gitee org issue list <org>` | `GET /orgs/{org}/issues` | List org issues |

---

### 3.7 `gitee search` — Search

**Corresponding Gitee API:** `/search/*`

| Subcommand | Gitee API | Description |
|------------|-----------|-------------|
| `gitee search repos <query>` | `GET /search/repositories` | Search repos |
| `gitee search issues <query>` | `GET /search/issues` | Search issues |
| `gitee search users <query>` | `GET /search/users` | Search users |

**`gitee search repos` Parameters:**

| Parameter | API Field | Description |
|-----------|-----------|-------------|
| `--language string` | `language` | Filter by programming language |
| `--sort string` | `sort` | `stars`\|`forks`\|`updated` |
| `--order string` | `order` | `asc`\|`desc` (default `desc`) |
| `--limit int` | `per_page` | Max results |

---

### 3.8 `gitee api` — Raw API Access

Execute Gitee API v5 requests directly, for debugging and automation scripts:

```bash
gitee api GET /user
gitee api GET /user/repos?type=owner
gitee api POST /user/repos --field name=my-repo --field private=true
gitee api DELETE /repos/owner/repo
gitee api PATCH /repos/owner/repo --field description="new desc"
```

**Parameters:**

| Parameter | Description |
|-----------|-------------|
| `<method>` | HTTP method: GET / POST / PUT / PATCH / DELETE |
| `<path>` | API path (e.g., `/user/repos`), auto-prefixed with `--api-url` |
| `--field key=value` | Request body parameter (JSON), can use multiple times |
| `--header key=value` | Custom request header, can use multiple times |
| `--raw-field key=value` | String type parameter (no JSON type inference) |
| `--input file` | Read request body from file (JSON) |
| `--include` | Output response headers |
| `--paginate` | Auto-paginate and merge JSON arrays |

---

### 3.9 `gitee config` — Configuration Management

| Subcommand | Description |
|------------|-------------|
| `gitee config list` | List all config (including source: file/env/default) |
| `gitee config set <key> <value>` | Set config item (write to `~/.gitee-cli/settings.json`) |
| `gitee config get <key>` | Read specific config item |
| `gitee config unset <key>` | Delete config item |

**Configurable Items:**

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `protocol` | string | `https` | git protocol (`ssh`\|`https`) |
| `editor` | string | `$EDITOR` | Commit message/body editor |
| `browser` | string | system default | Browser open command |
| `prompt` | string | `enabled` | Interactive prompt switch (`enabled`\|`disabled`) |
| `output` | string | `table` | Default output format |
| `pager` | string | `$PAGER` | Pager command for viewing |

---

### 3.10 `gitee completion` — Shell Completion

| Subcommand | Description |
|------------|-------------|
| `gitee completion bash` | Output Bash completion script |
| `gitee completion zsh` | Output Zsh completion script |
| `gitee completion fish` | Output Fish completion script |
| `gitee completion powershell` | Output PowerShell completion script |

**Installation Examples:**
```bash
# Bash
gitee completion bash >> ~/.bashrc

# Zsh
gitee completion zsh > "${fpath[1]}/_gitee"

# Fish
gitee completion fish > ~/.config/fish/completions/gitee.fish
```

---

## 4. Output Format Specification

### 4.1 Table Format (default `--output table`)

- Use monospace column alignment, UTF-8 encoding
- Column width auto-fits terminal width, longest column truncated to 60 chars (trailing `...`)
- Table header uppercase bold (ANSI `\033[1m`)
- Time fields show relative time (e.g., `2 days ago`)

### 4.2 JSON Format (`--output json`)

- Pretty JSON output (2 space indentation)
- `list` commands return JSON array
- `view` commands return single JSON object
- Error messages output to stderr (doesn't affect pipe)

### 4.3 CSV Format (`--output csv`)

- First row is header (lowercase field names)
- Values wrapped in double quotes, internal double quotes escaped as `""`
- Suitable for import to Excel/database

### 4.4 Pagination Output

- `--paginate`: Auto-traverse all pages, merge into single result set
- `--page N`: Specify page number (`1`-based)
- `--per-page N`: Items per page (max 100)
- `--limit N`: Limit total output items (priority over pagination, `0` = unlimited)

---

## 5. Error Handling Specification

| Scenario | Exit Code | stderr Output |
|----------|-----------|---------------|
| Not logged in | 1 | `Error: not authenticated. Run 'gitee auth login' first.` |
| Network timeout | 1 | `Error: failed to connect to gitee.com: <detail>` |
| API returns 401 | 1 | `Error: authentication failed. Token may be expired. Run 'gitee auth refresh'.` |
| API returns 403 | 1 | `Error: insufficient permissions: <message>` |
| API returns 404 | 1 | `Error: not found: <resource>` |
| API returns 422 | 1 | `Error: validation failed: <field errors>` |
| API returns 5xx | 1 | `Error: Gitee server error (HTTP <code>): <message>` |
| Parameter error | 2 | `Error: <description>` + usage help |
| Token expired | 1 | Prompt `gitee auth refresh` or `gitee auth login` |
| Operation cancelled | 1 | `Aborted.` |

---

## 6. Internationalization (i18n)

- Default language: Chinese (`zh-CN`)
- Support `--lang en` to switch to English output (or `GITEE_LANG=en`)
- Language packs in `internal/i18n/` directory (`zh_CN.go`, `en_US.go`)
- Error messages, help text, command descriptions all support i18n

---

## 7. Non-Functional Requirements

| Requirement | Value |
|-------------|-------|
| Startup time | < 100ms (`gitee --version` cold start) |
| First compile time | < 15s (Go 1.23 incremental compile) |
| Network timeout | Default 30s (`--timeout` can override) |
| Retry strategy | 5xx auto-retry 3 times (exponential backoff, max 8s) |
| Token cache | User info cached 5 minutes (avoid repeated `GET /user`) |
| Proxy | Respect `HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY` environment variables |
| Cross-platform | Linux amd64/arm64, macOS amd64/arm64 (M1), Windows amd64 |
| Binary size | < 20MB (enable `-ldflags="-s -w"` compression) |

---

## 8. Gitee API v5 Endpoint Coverage Matrix

| Feature Module | API Endpoints | gitee-cli Coverage |
|---------------|---------------|-------------------|
| Repos | 75+ | 8 core endpoints |
| Issue | 23 | 8 endpoints |
| Pull Request | 27 | 9 endpoints |
| Gist | 17 | 7 endpoints |
| Organization | 14 | 5 endpoints |
| User | 18 | via auth/search |
| Search | 3 | 3 endpoints (full coverage) |
| Webhook | 6 | Not covered yet (v2.0 plan) |
| Enterprise | 20+ | Not covered yet (v2.0 plan) |