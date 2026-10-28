# gitee-cli Functional Specification Document

> ⚠️ **Status notice (v0.13.2, 2026-10 maintenance round)**: This document is an early design
> draft; parts of it are **not implemented or out of date**. The authoritative behavior is the
> code, `gitee --help`, `gitee <cmd> --agent-help`, `gitee help reference` and
> `gitee help environment`. Paragraphs that diverge most from the current implementation are
> corrected inline.


Version: 1.0.0 (design draft; see the banner above for current-state corrections) | Language: Go 1.27+ | Reference: Gitee API v5 + GitHub `gh` CLI

---

## 1. Project Overview

### 1.1 Project Information

| Field | Value |
|-------|-------|
| Project Name | `gitee-cli` |
| Type | Command Line Tool (CLI) |
| Language | Go 1.27+ |
| Framework | Cobra (config is hand-written JSON, no Viper) |
| Goal | Become Gitee's official CLI tool, equivalent to GitHub `gh` |
| API Base | `https://gitee.com/api/v5` |
| Reference Projects | [`gh`](https://github.com/cli/cli), [`glab`](https://github.com/cli/go-gitlab) |

### 1.2 Core Features Summary

```
gitee auth login|logout|status|refresh|token        # PAT authentication (no OAuth2 device flow)
gitee repo list|create|view|fork|edit|delete|clone|browse|branches  # Repository management
gitee repo tag|collaborator|star|watch|file|commits|compare|protect|archive|rename|sync  # Repo extras
gitee issue list|create|view|edit|close|reopen|comment  # Issue management
gitee pr list|create|view|merge|review|diff|checks|comments|checkout|close|reopen|edit|ready|status  # Pull Request
gitee gist list|create|view|edit|delete|star|clone|comments  # Gist
gitee org list|view|members|repos|issues                   # Organization (plural subcommands)
gitee search issues|users                            # Search (repos returns empty; code/commits unsupported here)
gitee api <method> <path> [--field key=val]          # Raw API access
gitee config set|get|list|unset                      # Configuration
gitee completion bash|zsh|fish|powershell            # Shell completion
gitee label|milestone|release|webhook|notification|ssh-key|user|alias|extension|agent|status  # Other groups
```

The command surface is authoritative only in `gitee help reference` (per-group `--help` and
`--agent-help` are the source of truth).

---

## 2. Global Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--repo string` | string | Target repository (owner/repo); inferred from git remote when omitted | auto |
| `--hostname string` | string | Gitee server address (enterprise/private deployment) | `gitee.com` |
| `--token string` | string | Personal access token (PAT, priority over config) | from config |
| `--api-url string` | string | API root URL | `https://{hostname}/api/v5` |
| `-o, --output string` | string | Output format `table`\|`json` (**no csv; invalid values error out**) | `table` |
| `-P, --paginate` | bool | Auto-paginate through all results | `false` |
| `--page int` | int | Page number | `1` |
| `--per-page int` | int | Items per page (max 100) | `30` |
| `--limit int` | int | Max results (total cap across pages with `--paginate`; `0` = unlimited) | `30` |
| `--timeout int` | int | HTTP timeout in seconds | `30` |
| `-v, --verbose` | bool | Output HTTP request/response details | `false` |
| `--no-color` | bool | Disable ANSI color output | `false` |
| `--no-extensions` | bool | Skip loading external extensions | `false` |
| `--agent-help` | bool | Long, agent-targeted help (selected commands; `-o json` for structure) | `false` |

> The old design flags `--config` / `--no-interaction` / `--insecure` / `--lang` do **not** exist;
> the config file path is fixed at `~/.gitee-cli/settings.json`.

**Environment variables (13; run `gitee help environment`):**

| Variable | Mapping / meaning |
|----------|-------------------|
| `GITEE_TOKEN` | Access token, same as `--token` |
| `GITEE_HOSTNAME` | Same as `--hostname` |
| `GITEE_API_URL` | Same as `--api-url` |
| `GITEE_OUTPUT` | Default output format, same as `--output`; values other than `table`/`json` are rejected |
| `GITEE_VERBOSE` | Set to `1`/`true`/`yes`/`on` to enable HTTP details (stderr), same as `--verbose` |
| `GITEE_TIMEOUT` | HTTP timeout in seconds, same as `--timeout`; non-integers error out |
| `GITEE_LIMIT` | Default `--limit` value |
| `GITEE_PAGINATE` | Set to `1`/`true`/`yes`/`on` to enable `--paginate` by default |
| `GITEE_NO_EXTENSIONS` | Set to `1`/`true`/`yes`/`on` to skip extension loading |
| `NO_COLOR` | Non-empty disables colors, same as `--no-color` |
| `GITEE_ALIASES_FILE` | Override aliases file path |
| `GITEE_EXT_DIR` | Override extension directory |
| `BROWSER` | Browser binary used by `repo browse` |

> Proxy support comes from Go `net/http` standard variables (`HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY`);
> the CLI does not parse them itself.

---

## 3. Command Detailed Specifications

### 3.1 `gitee auth` — Authentication Management

**Mechanism: personal access token (PAT).** gitee-cli has **no** OAuth2 device flow, no browser
authorization and no `refresh_token` exchange. `auth login` reads an existing token and verifies
it against `GET /user`.

**Token resolution order:** `--token` flag > `GITEE_TOKEN` env var > `~/.gitee-cli/settings.json`.

| Subcommand | Behavior |
|------------|----------|
| `gitee auth login [--token <PAT>]` | No flag: read the existing token and verify `/user`; with `--token`: verify then persist to `~/.gitee-cli/settings.json` (dir 0700 / file 0600) |
| `gitee auth logout` | Remove the stored token |
| `gitee auth status` | `GET /user`, show login state and username |
| `gitee auth refresh` | Stub: prints "Token refresh is not yet supported. Please re-authenticate with 'gitee auth login'." and exits 0; it refreshes nothing and is **not** an OAuth `refresh_token` flow |
| `gitee auth token` | Print the current access token (for piping) |

**Real `gitee auth login` flow:**
```
1. Resolve the candidate token (--token > GITEE_TOKEN > settings.json)
2. GET /user with header `Authorization: token <PAT>` → verify + fetch username
3. Only when --token was passed, write it to ~/.gitee-cli/settings.json (0600)
```
> ⚠️ A `--token` value is visible via `ps` and lands in shell history; prefer `GITEE_TOKEN`
> outside one-off scripts.

**`gitee auth login` flags:**

| Flag | Description |
|------|-------------|
| `-t, --token string` | Use a specific token; saved after successful verification |

> The old draft's `--with-token` (stdin) and `--scopes` do **not** exist.

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
| `--sort string` | `created`\|`updated`\|`pushed`\|`full_name`\|`stars` | `full_name` |
| `--direction string` | `asc`\|`desc` | `asc` |
| `-L, --limit int` | Max repositories to return | `30` |

> The old draft's `--type` does not exist; the command takes no positional argument (use `--repo`
> or let it infer from the git remote).

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
| `LABELS` | `labels[].name` | Labels (comma-joined; `-` when none) |
| `UPDATED` | `updated_at` | Update date (`YYYY-MM-DD`) |

> The table has exactly these 5 columns; there is no `ASSIGNEE` column.

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
| `--state string` | `state` | `open` (`open`\|`closed`\|`merged`\|`all`) |
| `--head string` | `head` | — |
| `--base string` | `base` | — |

> The old draft's `--author` / `--sort` / `--direction` do not exist on `pr list`.

**`gitee pr merge` Parameters:**

| Parameter | Description |
|-----------|-------------|
| `--squash` | Squash merge (body `squash=true`) |
| `--squash-message string` | Commit message for the squash merge (body `squash_message`) |

> The old draft's `--rebase` / `--merge-commit-message` / `--delete-branch` do **not** exist:
> the CLI only implements the squash and squash-message merge options.

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
| `--file stringArray` | `files` | `filename=content` or `filename=@path`, repeatable |
| `--filename string` | — | Filename in stdin mode (content read from stdin) |
| `-i, --interactive` | — | Interactive editing mode |

> There is no `--from-stdin`; for stdin input use `--filename <name>` (optionally with `--file`).

**`gitee gist list` Output Fields:**

| Field | API Field | Description |
|-------|-----------|-------------|
| `ID` | `id` | Gist ID |
| `DESCRIPTION` | `description` | Description (truncated to 50 chars, `-` if empty) |
| `PUBLIC` | `public` | Whether public |
| `FILES` | `files` count | Number of files |
| `UPDATED` | `updated_at` | Update time |

---

### 3.6 `gitee org` — Organization Management

**Corresponding Gitee API:** `/orgs`, `/orgs/{org}`

| Subcommand | Gitee API | Description |
|------------|-----------|-------------|
| `gitee org list` | `GET /user/orgs` | List user's organizations |
| `gitee org view <org>` | `GET /orgs/{org}` | View org details |
| `gitee org members <org>` | `GET /orgs/{org}/members` | List org members |
| `gitee org repos <org>` | `GET /orgs/{org}/repos` | List org repos |
| `gitee org issues <org>` | `GET /orgs/{org}/issues` | List org issues |

---

### 3.7 `gitee search` — Search

**Corresponding Gitee API:** `/search/*`

| Subcommand | Gitee API | Description |
|------------|-----------|-------------|
| `gitee search repos <query>` | `GET /search/repositories` | Search repos |
| `gitee search issues <query>` | `GET /search/issues` | Search issues |
| `gitee search users <query>` | `GET /search/users` | Search users |
| `gitee search code <query>` | `GET /search/code` | Code search (404 on this Gitee instance; CLI explains and exits non-zero) |
| `gitee search commits <query>` | `GET /search/commits` | Commit search (404 on this Gitee instance; CLI explains and exits non-zero) |

> ⚠️ Current state (measured on v0.13.2): `search issues` / `search users` work; `search repos`
> returns empty for every query (upstream API behavior change; CLI prints `(no results)`);
> `search code` / `search commits` are unavailable on this instance.

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

### 4.3 CSV (**not supported**)

`-o/--output` accepts only `table` and `json`; any other value (including the `csv` claimed by
early docs) errors out instead of silently falling back to table. Use `-o json` + `jq` when you
need tabular data (see `gitee help formatting`).

### 4.4 Pagination Output

- `--paginate`: Auto-traverse all pages, merge into single result set
- `--page N`: Specify page number (`1`-based)
- `--per-page N`: Items per page (max 100)
- `--limit N`: Limit total output items (priority over pagination, `0` = unlimited)

---

## 5. Error Handling Specification

| Scenario | Exit Code | stderr Output (`Error: <operation>: <detail>`) |
|----------|-----------|---------------|
| Not logged in (local check, no request) | 1 | `not authenticated. Set GITEE_TOKEN or run 'gitee auth login'` |
| Network error / timeout | 1 | `<transport error>` (e.g. `context deadline exceeded`, `connection refused`) |
| API returns 401 | 1 | `HTTP 401: <Gitee message>`; falls back to `HTTP 401: authentication failed: run 'gitee auth login' or set GITEE_TOKEN` when the body has no message |
| API returns 403 | 1 | `HTTP 403: <Gitee message>`, or `HTTP 403: permission denied` |
| API returns 404 | 1 | `HTTP 404: <Gitee message>`, or `HTTP 404: resource does not exist or you lack read access` |
| API returns 422 | 1 | `HTTP 422: <Gitee message>`, or `HTTP 422: validation failed` |
| API returns 5xx | 1 | `HTTP 5xx: <Gitee message>`, or `internal server error` / `service unavailable` |
| Parameter error | 1 | `unknown flag: --x` / `unknown command "x" for "gitee ..."` (Cobra output, also prefixed with `Error: `) |
| Operation cancelled (prompt declined) | 0 | Prints `Cancelled`; nothing is performed |

---

## 6. Internationalization (i18n)

- **Not implemented.** There is no `--lang`, no language-pack directory and no runtime language
  switching; command help and output are fixed strings.
- Multilingual support exists only at the documentation layer: `README.md` / `README_en.md` /
  `README_ja.md` and `docs/{zh,en,ja}/`.

---

## 7. Non-Functional Requirements

| Requirement | Value |
|-------------|-------|
| Startup time | < 100ms (`gitee --version` cold start) |
| First compile time | < 20s (Go 1.27 incremental compile; machine-dependent) |
| Network timeout | Default 30s (`--timeout` can override) |
| Retry strategy | **GET only**: 5xx/network errors retried 3 times (starting at 2s, max 8s); writes are never auto-retried |
| Username source | Read cached username from `settings.json`, fall back to `GET /user` when missing (no TTL cache) |
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
| Webhook | 6 | Implemented (create / list / view / edit / delete / ping) |
| Enterprise | 20+ | Private deployments via `--hostname` / `GITEE_HOSTNAME` |