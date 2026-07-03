package cmd

// rootAgentHelp is the agent-targeted help printed when the user runs
// `gitee --agent-help`. It is intentionally longer and more structured than
// the default `--help` output: agents read carefully, humans skim.
//
// Coverage table below should be updated whenever a new command gains
// agent-help support. Format: `gitee <full-path> --agent-help` then a status
// marker (✅ / 🆕 / ❌). This makes it easy for agents to discover what's
// available without trying every command.
const rootAgentHelp = `# gitee — Agent Help (root overview)

## What this is
gitee-cli ships TWO help systems side-by-side:

  --help         Short, friendly, optimized for humans skimming docs.
  --agent-help   Long, structured, optimized for agents reading carefully.

--agent-help content includes everything in --help PLUS:
  - Pre-conditions (what must be true before you can run the command)
  - Post-conditions (what is true after a successful run)
  - Idempotency (can you re-run safely? what happens if you do?)
  - Atomicity (are there partial-success states?)
  - Side effects (emails, webhooks, notifications)
  - Common error codes + how to recover
  - OAuth scope required
  - Rate limit
  - Race conditions and how to avoid them
  - Next-step orchestration hints (what command should you run after this?)
  - Parsing hints for stdout (machine-friendly examples)

## When to use --agent-help
- Before writing a new agent workflow that touches Gitee
- When a command fails and you don't know why
- When you need to know the side effects of an action before triggering it
- When you need to chain commands together in a multi-step workflow

## How to invoke
  gitee <command> [subcommand] --agent-help

## Coverage matrix (v0.11.0)

### repo — Repository management
  gitee repo list           --agent-help   ✅ available  🆕 v0.11.0
  gitee repo create         --agent-help   ✅ available  🆕 v0.11.0
  gitee repo view           --agent-help   ✅ available  🆕 v0.11.0
  gitee repo fork           --agent-help   ✅ available  🆕 v0.11.0
  gitee repo edit           --agent-help   ✅ available  🆕 v0.11.0
  gitee repo delete         --agent-help   ✅ available  🆕 v0.11.0
  gitee repo clone          --agent-help   ✅ available  🆕 v0.11.0
  gitee repo browse         --agent-help   ✅ available  🆕 v0.11.0
  gitee repo branches       --agent-help   ✅ available  🆕 v0.11.0
  gitee repo tag            --agent-help   ✅ available  🆕 v0.12.0
  gitee repo tag list       --agent-help   ✅ available  🆕 v0.12.0
  gitee repo tag create     --agent-help   ✅ available  🆕 v0.12.0
  gitee repo collaborator            --agent-help   ✅ available  🆕 v0.12.0
  gitee repo collaborator list       --agent-help   ✅ available  🆕 v0.12.0
  gitee repo collaborator add        --agent-help   ✅ available  🆕 v0.12.0
  gitee repo collaborator remove     --agent-help   ✅ available  🆕 v0.12.0
  gitee repo archive        --agent-help   ✅ available  🆕 v0.5.0
  gitee repo sync           --agent-help   ✅ available  🆕 v0.5.0
  gitee repo rename         --agent-help   ✅ available  🆕 v0.5.0
  gitee repo set-default    --agent-help   n/a  (operational/config — use --help)

### issue — Issue management
  gitee issue list          --agent-help   ✅ available  🆕 v0.11.0
  gitee issue create        --agent-help   ✅ available
  gitee issue view          --agent-help   ✅ available  🆕 v0.11.0
  gitee issue edit          --agent-help   ✅ available  🆕 v0.11.0
  gitee issue close         --agent-help   ✅ available  🆕 v0.11.0
  gitee issue reopen        --agent-help   ✅ available  🆕 v0.11.0
  gitee issue comment       --agent-help   ✅ available  🆕 v0.11.0
  gitee issue comments      --agent-help   ✅ available  🆕 v0.8.0
  gitee issue status        --agent-help   ✅ available  🆕 v0.10.0
  gitee issue label         --agent-help   ❌ not yet (legacy path; use 'gitee label')

### pr — Pull Request management
  gitee pr list             --agent-help   ✅ available  🆕 v0.11.0
  gitee pr create           --agent-help   ✅ available
  gitee pr view             --agent-help   ✅ available  🆕 v0.11.0
  gitee pr merge            --agent-help   ✅ available  🆕 v0.11.0
  gitee pr review           --agent-help   ✅ available  🆕 v0.11.0
  gitee pr close            --agent-help   ✅ available  🆕 v0.11.0
  gitee pr reopen           --agent-help   ✅ available  🆕 v0.11.0
  gitee pr diff             --agent-help   ✅ available  🆕 v0.11.0
  gitee pr comments         --agent-help   ✅ available  🆕 v0.11.0
  gitee pr comment          --agent-help   ✅ available  🆕 v0.8.0
  gitee pr checkout         --agent-help   ✅ available  🆕 v0.5.0
  gitee pr ready            --agent-help   ✅ available  🆕 v0.8.0
  gitee pr edit             --agent-help   ✅ available  🆕 v0.8.0
  gitee pr checks           --agent-help   ✅ available  🆕 v0.8.0
  gitee pr status           --agent-help   ✅ available  🆕 v0.10.0

### gist — Code snippets
  gitee gist list           --agent-help   ✅ available  🆕 v0.11.0
  gitee gist view           --agent-help   ✅ available  🆕 v0.11.0
  gitee gist create         --agent-help   ✅ available  🆕 v0.11.0
  gitee gist edit           --agent-help   ✅ available  🆕 v0.11.0
  gitee gist delete         --agent-help   ✅ available  🆕 v0.11.0
  gitee gist star           --agent-help   ✅ available  🆕 v0.11.0
  gitee gist unstar         --agent-help   ✅ available  🆕 v0.11.0
  gitee gist clone          --agent-help   ✅ available  🆕 v0.6.0

### ssh-key — SSH key management  🆕 v0.6.0
  gitee ssh-key list        --agent-help   ✅ available
  gitee ssh-key add         --agent-help   ✅ available
  gitee ssh-key delete      --agent-help   ✅ available

### gpg-key — GPG key management  ⚠  removed 2026-06
  gitee gpg-key *           — Gitee v5 API does not expose a GPG key endpoint
  Manage GPG keys via https://gitee.com/profile/gpg_settings

### webhook — Repository webhooks  🆕 v0.7.0
  gitee webhook list        --agent-help   ✅ available
  gitee webhook create      --agent-help   ✅ available
  gitee webhook view        --agent-help   ✅ available
  gitee webhook edit        --agent-help   ✅ available
  gitee webhook delete      --agent-help   ✅ available
  gitee webhook test        --agent-help   ✅ available

### org — Organization management
  gitee org list            --agent-help   ✅ available  🆕 v0.11.0
  gitee org view            --agent-help   ✅ available  🆕 v0.11.0
  gitee org member          --agent-help   ✅ available  🆕 v0.11.0
  gitee org repo            --agent-help   ✅ available  🆕 v0.11.0
  gitee org issue           --agent-help   ✅ available  🆕 v0.11.0

### agent — Agent onboarding  🆕 v0.11.0
  gitee agent onboard       --agent-help   ✅ available  🆕 v0.11.0

### release — Release management  🆕 v0.5.0
  gitee release list        --agent-help   ✅ available
  gitee release view        --agent-help   ✅ available
  gitee release create      --agent-help   ✅ available
  gitee release edit        --agent-help   ✅ available
  gitee release delete      --agent-help   ✅ available
  gitee release download    --agent-help   ✅ available
  gitee release upload      --agent-help   ✅ available  🆕 v0.10.0
  gitee release delete-asset --agent-help  ✅ available  🆕 v0.10.0

### label — Label management  🆕 v0.5.0 (top-level)
  gitee label list          --agent-help   ✅ available
  gitee label view          --agent-help   ✅ available
  gitee label create        --agent-help   ✅ available
  gitee label edit          --agent-help   ✅ available
  gitee label delete        --agent-help   ✅ available
  gitee label clone         --agent-help   ✅ available

### milestone — Repository milestones  🆕 v0.12.0
  gitee milestone list      --agent-help   ✅ available
  gitee milestone view      --agent-help   ✅ available
  gitee milestone create    --agent-help   ✅ available
  gitee milestone edit      --agent-help   ✅ available
  gitee milestone delete    --agent-help   ✅ available

### notification — Notification center  🆕 v0.12.0
  gitee notification list   --agent-help   ✅ available
  gitee notification count  --agent-help   ✅ available
  gitee notification read   --agent-help   ✅ available

### api — Raw API access
  gitee api                 --agent-help   ❌ not yet (escape hatch — see --help)

### status — Cross-repo activity summary  🆕 v0.9.0
  gitee status              --agent-help   ✅ available  🆕 v0.9.0

### extension — gitee-cli extension management  🆕 v0.9.0
  gitee extension list      --agent-help   ✅ available  🆕 v0.9.0
  gitee extension install   --agent-help   ✅ available  🆕 v0.9.0
  gitee extension remove    --agent-help   ✅ available  🆕 v0.9.0
  gitee extension upgrade   --agent-help   ✅ available  🆕 v0.9.0
  gitee extension create    --agent-help   ✅ available  🆕 v0.9.0
  External extensions (from ~/.gitee-cli/extensions/) are loaded at startup
  via ` + "`" + `gitee <name>` + "`" + `; --no-extensions disables that loader.

### alias — User-defined command shortcuts  🆕 v0.9.0
  gitee alias set           --agent-help   ✅ available  🆕 v0.9.0
  gitee alias list          --agent-help   ✅ available  🆕 v0.9.0
  gitee alias delete        --agent-help   ✅ available  🆕 v0.9.0
  Aliases (from ~/.gitee-cli/aliases.json) are expanded at startup: typing
  ` + "`" + `gitee co 123` + "`" + ` after ` + "`" + `gitee alias set co 'pr checkout'` + "`" + ` is rewritten to
  ` + "`" + `gitee pr checkout 123` + "`" + ` before cobra parses argv. Built-in commands
  always win over aliases of the same name.

### help topics — Cross-cutting documentation  🆕 v0.9.0
  gitee help environment    --agent-help   n/a  (use 'gitee help environment')
  gitee help exit-codes     --agent-help   n/a  (use 'gitee help exit-codes')
  gitee help formatting     --agent-help   n/a  (use 'gitee help formatting')
  gitee help reference      --agent-help   n/a  (auto-generated command index)
  These replace gh-style 'man pages' for things that don't fit under a single
  command. The reference page is generated from the live cobra tree at run time.

### search — Search (code + commits are 🆕 v0.10.0)
  gitee search repos        --agent-help   ✅ available  🆕 v0.11.0
  gitee search issues       --agent-help   ✅ available  🆕 v0.11.0
  gitee search users        --agent-help   ✅ available  🆕 v0.11.0
  gitee search code         --agent-help   ✅ available  🆕 v0.10.0
  gitee search commits      --agent-help   ✅ available  🆕 v0.10.0

### auth, config, completion, version
  These are operational / config commands. --agent-help is not needed for
  them; consult --help.
  gitee auth switch         --agent-help   n/a  (operational/config — use --help)

## Roadmap
  v0.4.0  ✅  Initial release: issue create, pr create
  v0.4.1  ✅  Bug fixes + expand to other "write" commands
  v0.5.0  ✅  P0 commands: release, label (top-level), pr checkout, repo archive/sync/rename
  v0.5.0       Add stdout "next steps" hints on success (P2)
  v0.6.0  ✅  P1a: ssh-key, gpg-key, gist clone
  v0.6.0       Add agent identity signature in body (P1)
  v0.7.0  ✅  P1b: webhook (list/create/view/edit/delete/test)
  v0.8.0  ✅  P1c: issue comments list, pr ready/comment/edit/checks
  v0.9.0  ✅  P2a: status (issues assigned / PRs awaiting review)
  v0.9.0  ✅  P2b: extension (5 verb + dynamic load + --no-extensions flag)
  v0.9.0  ✅  P2c: alias (set/list/delete + dynamic expansion)
  v0.9.0  ✅  Help topics: environment / exit-codes / formatting / reference
  v0.10.0 ✅  8 small features: pr/issue status, release upload/delete-asset,
                search code/commits, auth switch, repo set-default
  v0.11.0 ✅  Expand --agent-help to repo/issue/pr/gist/org/search coverage
                (full read/write/agent surface)
  v0.12.0 ✅  P0 gaps: repo tag (list/create), issue edit/close/reopen,
                milestone CRUD, notification center
  v0.9.0       P3: pipeline (Gitee Go)

## Design philosophy
gitee-cli is built on the principle that CLIs increasingly have TWO users:
humans (who want brevity) and AI agents (who want completeness). Sharing
a single --help system is a false economy that hurts both audiences.
Splitting the help systems costs a small amount of maintenance but
unlocks dramatically better agent reliability and discoverability.

If you maintain another CLI: consider adding --agent-help. It's a few
hours of work, a few hundred lines of structured text, and a real
improvement for any tool that AI agents consume.
`
