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

## Coverage matrix (v0.4.0)

### repo — Repository management
  gitee repo list           --agent-help   ❌ not yet
  gitee repo create         --agent-help   ❌ not yet
  gitee repo view           --agent-help   ❌ not yet
  gitee repo fork           --agent-help   ❌ not yet
  gitee repo edit           --agent-help   ❌ not yet
  gitee repo delete         --agent-help   ❌ not yet
  gitee repo clone          --agent-help   ❌ not yet
  gitee repo browse         --agent-help   ❌ not yet
  gitee repo branches       --agent-help   ❌ not yet

### issue — Issue management
  gitee issue list          --agent-help   ❌ not yet
  gitee issue create        --agent-help   ✅ available
  gitee issue view          --agent-help   ❌ not yet
  gitee issue edit          --agent-help   ❌ not yet
  gitee issue close         --agent-help   ❌ not yet
  gitee issue reopen        --agent-help   ❌ not yet
  gitee issue comment       --agent-help   ❌ not yet
  gitee issue label         --agent-help   ❌ not yet

### pr — Pull Request management
  gitee pr list             --agent-help   ❌ not yet
  gitee pr create           --agent-help   ✅ available
  gitee pr view             --agent-help   ❌ not yet
  gitee pr merge            --agent-help   ❌ not yet
  gitee pr review           --agent-help   ❌ not yet
  gitee pr close            --agent-help   ❌ not yet
  gitee pr reopen           --agent-help   ❌ not yet
  gitee pr diff             --agent-help   ❌ not yet
  gitee pr comments         --agent-help   ❌ not yet

### gist — Code snippets
  All subcommands           --agent-help   ❌ not yet

### org — Organization management
  All subcommands           --agent-help   ❌ not yet

### api — Raw API access
  gitee api                 --agent-help   ❌ not yet (escape hatch — see --help)

### auth, config, search, completion, version
  These are operational / config commands. --agent-help is not needed for
  them; consult --help.

## Roadmap
  v0.4.0  ✅  Initial release: issue create, pr create
  v0.4.1       Bug fixes + expand to other "write" commands
  v0.5.0       Expand agent-help to all commands
  v0.5.0       Add stdout "next steps" hints on success (P2)
  v0.6.0       Add agent identity signature in body (P1)

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
