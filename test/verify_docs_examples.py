#!/usr/bin/env python3
"""Verify every runnable `gitee ...` example in README*.md and docs/**/*.md.

P1 acceptance item A. The checker parses fenced code blocks and inline code
spans, extracts every invocation of the `gitee` binary, and validates it
against the *live* cobra command tree of a built binary:

  * command path exists (and is not just a command *group* that only
    prints its own help when handed a stray positional),
  * every long flag / shorthand is actually defined on that command,
  * the command is a leaf, or every extra token after a group is a real
    subcommand.

Known non-examples are deliberately NOT reported (see is_placeholder /
split_commands): `gitee api GET /user` (api legitimately takes method
positionals), global flags placed before the subcommand
(`gitee --verbose repo list`), `&&` chains, `|` pipelines, `>` redirects,
`$(...)` substitutions, `gitee <command>` placeholders, and documentation
shorthand such as `gitee auth login|logout|status`.

Exit codes: 0 = all examples valid; 1 = at least one real finding;
2 = setup problem (binary missing / unparsable help).

Usage:
    python3 test/verify_docs_examples.py [--binary PATH] [--root DIR]
                                        [--json] [--verbose] [--limit N]
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shlex
import subprocess
import sys

# Commands that have subcommands *and* legitimately accept positional
# arguments (their own RunE handles them). Everything else that has
# subcommands is treated as a pure group: `gitee <group> <word>` where
# <word> is not a subcommand is a documentation bug.
ARG_ACCEPTING_GROUPS = {"api"}

# Tokens that make an example a template/placeholder rather than a command
# someone can paste. Such examples are skipped, not validated.
PLACEHOLDER_RE = re.compile(r"[<>{}]|\.\.\.|\[[A-Za-z][A-Za-z ]*\]|\$[A-Za-z_][A-Za-z0-9_]*")

# `gitee` must be followed by whitespace (so gitee.com, gitee-cli,
# gitee/gitee-cli are never matched).
GITEE_RE = re.compile(r"(?<![\w./-])gitee(?:\.exe)?(?=\s)")


def die(msg: str) -> "None":
    print(f"verify_docs_examples: ERROR: {msg}", file=sys.stderr)
    raise SystemExit(2)


# ---------------------------------------------------------------------------
# Markdown scanning
# ---------------------------------------------------------------------------
def iter_code_examples(path: str):
    """Yield (lineno, text, kind) for fenced-block lines and inline spans."""
    try:
        with open(path, encoding="utf-8", errors="replace") as fh:
            lines = fh.read().splitlines()
    except OSError as exc:  # pragma: no cover - unreadable file
        print(f"warning: cannot read {path}: {exc}", file=sys.stderr)
        return

    fence = None
    i = 0
    while i < len(lines):
        line = lines[i]
        stripped = line.lstrip()
        m = re.match(r"^(```+|~~~+)", stripped)
        if m:
            marker = m.group(1)
            if fence is None:
                fence = marker[0] * 3
                i += 1
                continue
            if stripped.startswith(fence):
                fence = None
                i += 1
                continue
        if fence is not None:
            # Join shell line-continuations so multi-line commands are one logic line.
            start_no = i + 1
            text = line
            while text.rstrip().endswith("\\") and i + 1 < len(lines):
                text = text.rstrip()[:-1] + " " + lines[i + 1]
                i += 1
            yield start_no, text, "fenced"
        else:
            for span in re.finditer(r"`([^`\n]+)`", line):
                yield i + 1, span.group(1), "inline"
        i += 1


def strip_comment(text: str) -> str:
    """Drop a trailing `# comment` (shell style) outside quotes."""
    out = []
    quote = None
    prev = ""
    for ch in text:
        if quote:
            if ch == quote:
                quote = None
            out.append(ch)
            prev = ch
            continue
        if ch in ("'", '"'):
            quote = ch
            out.append(ch)
        elif ch == "#" and prev in ("", " ", "\t"):
            break
        else:
            out.append(ch)
            prev = ch
    return "".join(out)


# Words that may legitimately precede the gitee binary on a command line.
# Anything else (e.g. `-o`, `mv`) means the matched token is a *filename*
# argument, not an invocation (`go build -o gitee ./cmd/gitee`).
COMMAND_PREFIX_WORDS = {
    "sudo", "doas", "env", "time", "exec", "command", "nohup", "xargs",
    "watch", "eval", "sh", "bash", "zsh",
}


def is_command_position(text: str, start: int) -> bool:
    prefix = text[:start].rstrip()
    if not prefix:
        return True
    words = prefix.split()
    last = words[-1]
    if last in COMMAND_PREFIX_WORDS:
        return True
    if last.endswith("$(") or last in {"$(", "`", '"', "'", "(", "|", ";", "&&", "||"}:
        return True
    return False


def iter_gitee_segments(text: str):
    """Yield shell-ish segments that begin with the gitee binary."""
    text = strip_comment(text)
    for m in GITEE_RE.finditer(text):
        start = m.start()
        if not is_command_position(text, start):
            continue
        # A segment ends at an unquoted control operator / redirect / closing
        # substitution. Trim the leading text before `gitee`.
        end = len(text)
        quote = None
        idx = m.end()
        while idx < len(text):
            ch = text[idx]
            if quote:
                if ch == quote:
                    quote = None
            elif ch in ("'", '"'):
                quote = ch
            elif ch in ";|&)`>":
                # `&` only ends a command when followed by `&`; a lone `&`
                # redirects the command, which we treat as an end too.
                if ch == "&" and idx + 1 < len(text) and text[idx + 1] == "&":
                    end = idx
                    break
                if ch == ">":
                    end = idx
                    break
                if ch in ";|)`":
                    end = idx
                    break
            idx += 1
        seg = text[start:end].rstrip()
        if seg:
            yield seg


def tokenize(seg: str):
    try:
        return shlex.split(seg, posix=True)
    except ValueError:
        return None


# ---------------------------------------------------------------------------
# Live command tree (from the binary itself)
# ---------------------------------------------------------------------------
class CommandTree:
    def __init__(self, binary: str):
        self.binary = binary
        self.root_flags = {"long": set(), "short": set(), "arity": {}, "short_arity": {}}
        self.children = {}  # tuple(path) -> dict(child name -> None)
        self._help_cache = {}
        self._build()

    def _run(self, args):
        return subprocess.run(
            [self.binary] + args,
            capture_output=True,
            text=True,
            timeout=60,
            env={**os.environ, "NO_COLOR": "1", "CLICOLOR": "0"},
        )

    def _build(self):
        self._parse_help(())
        reference = self._run(["help", "reference"])
        if reference.returncode != 0:
            die(f"`{self.binary} help reference` failed (exit {reference.returncode})")
        paths = []
        for line in reference.stdout.splitlines():
            m = re.match(r"^(gitee (?:\S+)(?: \S+)*?)(?:\s{2,})(.*)$", line)
            if m:
                paths.append(m.group(1)[len("gitee "):])
        # `gitee --help` is authoritative for top-level commands (it includes
        # `help` and `version`, which the generated reference may omit), while
        # `help reference` enumerates the deeper leaves.
        for name in self._parse_help(()).get("children", {}):
            self.children.setdefault((), {})[name] = None
        for path in dict.fromkeys(paths):
            parts = tuple(path.split())
            for depth in range(1, len(parts) + 1):
                parent = self.children.setdefault(parts[:depth - 1], {})
                parent[parts[depth - 1]] = None

    def _parse_help(self, path):
        key = tuple(path)
        if key in self._help_cache:
            return self._help_cache[key]
        proc = self._run([*path, "--help"])
        if proc.returncode != 0:
            die(f"`{' '.join([self.binary, *path])} --help` failed")
        text = proc.stdout
        info = {
            "long": set(),
            "short": set(),
            "arity": {},
            "short_arity": {},
            "children": {},
            "usage_lines": [],
        }
        section = None
        for line in text.splitlines():
            if re.match(r"^Available Commands:", line):
                section = "children"
                continue
            if re.match(r"^Flags:", line) or re.match(r"^Global Flags:", line):
                section = "flags"
                continue
            if re.match(r"^Usage:", line):
                section = "usage"
                continue
            if re.match(r"^\S", line):  # a new top-level heading
                section = None
                continue
            if section == "children":
                m = re.match(r"^\s\s([\w.-]+)\s{2,}", line)
                if m:
                    info["children"][m.group(1)] = None
            elif section == "flags":
                # `  -o, --output string   desc` / `      --repo string  desc`
                m = re.match(r"^\s\s+(?:-(\w), )?--([\w.-]+)(?: ([\w\[\]]+))?\s{2,}", line)
                if m:
                    short, long, typ = m.group(1), m.group(2), m.group(3)
                    info["long"].add(long)
                    if short:
                        info["short"].add(short)
                    takes_value = bool(typ)
                    info["arity"][long] = takes_value
                    if short:
                        info["short_arity"][short] = takes_value
            elif section == "usage":
                if line.strip():
                    info["usage_lines"].append(line.strip())
        self._help_cache[key] = info
        return info

    def root(self):
        return self._parse_help(())


# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------
def validate_command(tree: CommandTree, tokens):
    """Return (findings, skipped_reason). findings is a list of reasons."""
    findings = []
    if not tokens:
        return findings, "empty"
    if tokens[0].split("/")[-1] not in ("gitee", "gitee.exe"):
        return findings, "not-gitee"
    rest = tokens[1:]
    if not rest:
        return findings, None

    for tok in rest:
        if PLACEHOLDER_RE.search(tok) or tok in ("command", "subcommand"):
            return findings, "placeholder"

    root_help = tree.root()
    i = 0
    # Global flags may be placed before the subcommand (`gitee --verbose repo list`).
    while i < len(rest) and rest[i].startswith("-"):
        tok = rest[i]
        if tok.startswith("--"):
            name, has_eq = (tok[2:].split("=", 1) + [None])[:2] if "=" in tok else (tok[2:], None)
            if name not in root_help["long"]:
                findings.append(f"unknown global flag --{name}")
                return findings, None
            i += 1 if has_eq is not None or not root_help["arity"].get(name) else 2
        else:
            ch = tok[1:2]
            if not ch or ch not in root_help["short"]:
                findings.append(f"unknown global shorthand {tok}")
                return findings, None
            takes = root_help["short_arity"].get(ch, False)
            i += 2 if (takes and tok == "-" + ch and i + 1 < len(rest)) else 1
    if i >= len(rest):
        return findings, None

    # Greedy longest-match of the command path, allowing flags in between.
    path = []
    node_children = tree.children.get((), {})
    while i < len(rest):
        tok = rest[i]
        if tok.startswith("-"):
            break
        if tok not in node_children:
            break
        path.append(tok)
        node_children = tree.children.get(tuple(path), {})
        i += 1

    path_key = tuple(path)
    info = tree._parse_help(path_key)
    # Merge subcommands discovered from --help so the tree is self-healing
    # even if `help reference` is incomplete.
    if info["children"]:
        node = tree.children.setdefault(path_key, {})
        for child in info["children"]:
            node.setdefault(child, None)

    # No subcommand matched at all -> unknown command / stray argument.
    if not path:
        return [f"unknown command `{rest[i].split()[0]}` (gitee has no such subcommand)"], None

    if path_key in [(p) for p in [(tuple(x.split())) for x in ARG_ACCEPTING_GROUPS]]:
        allow_positionals = True
    else:
        allow_positionals = False

    remaining = rest[i:]
    positional_seen = None
    k = 0
    while k < len(remaining):
        tok = remaining[k]
        if tok == "--":
            break
        if tok.startswith("--"):
            name = tok[2:].split("=", 1)[0]
            if name not in info["long"]:
                findings.append(
                    f"unknown flag --{name} for `gitee {' '.join(path)}`"
                )
                k += 1
                continue
            has_inline = "=" in tok
            k += 1 if (has_inline or not info["arity"].get(name)) else 2
            continue
        if tok.startswith("-") and len(tok) > 1 and not tok[1].isdigit():
            ch = tok[1]
            if ch not in info["short"]:
                findings.append(
                    f"unknown shorthand -{ch} for `gitee {' '.join(path)}`"
                )
                k += 1
                continue
            takes = info["short_arity"].get(ch, False)
            k += 2 if (takes and tok == "-" + ch and k + 1 < len(remaining)) else 1
            continue
        # positional
        if positional_seen is None:
            positional_seen = tok
        k += 1

    if positional_seen is not None and not allow_positionals and info["children"]:
        # `gitee issue close/reopen/edit` is documentation shorthand for
        # several sibling subcommands, not a single invocation.
        alternatives = positional_seen.split("/")
        if len(alternatives) > 1 and all(alt in info["children"] for alt in alternatives):
            return findings, "shorthand"
        # `gitee repo star owner/repo`: the group only prints help / errors.
        if positional_seen not in info["children"]:
            child_names = ", ".join(sorted(info["children"]))
            findings.append(
                f"`gitee {' '.join(path)}` is a command group (subcommands: {child_names}); "
                f"`{positional_seen}` is not one of them"
            )
        else:
            findings.append(
                f"`gitee {' '.join(path)} {positional_seen}` resolves to a subcommand but the "
                f"example places it after a group token"
            )
    return findings, None


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--binary", default=os.environ.get("GITEE_BIN"))
    ap.add_argument("--root", default=None)
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--verbose", action="store_true")
    ap.add_argument("--limit", type=int, default=200)
    args = ap.parse_args()

    root = os.path.abspath(args.root or os.path.join(os.path.dirname(__file__), ".."))
    # No implicit /tmp fallback: the caller must point at the binary it built,
    # otherwise a stale binary would silently validate the wrong tree.
    binary = args.binary
    if not binary:
        die("--binary (or GITEE_BIN) is required; build the current tree with "
            "`go build -o /tmp/gitee-verify ./cmd/gitee` first")
    probe = subprocess.run([binary, "version"], capture_output=True, text=True, timeout=30)
    if probe.returncode != 0:
        die(f"{binary} is not a working gitee binary (version exited {probe.returncode})")

    tree = CommandTree(binary)

    files = []
    for name in sorted(os.listdir(root)):
        if name.startswith("README") and name.endswith(".md"):
            files.append(os.path.join(root, name))
    docs_dir = os.path.join(root, "docs")
    for dirpath, _dirnames, filenames in os.walk(docs_dir):
        for name in sorted(filenames):
            if name.endswith(".md"):
                files.append(os.path.join(dirpath, name))
    files = [f for f in files if os.path.isfile(f)]
    if not files:
        die(f"no README*/docs markdown found under {root}")

    findings = []
    checked = 0
    checked_by_kind = {"fenced": 0, "inline": 0}
    skipped = {}
    spans = 0
    for path in files:
        rel = os.path.relpath(path, root)
        for lineno, text, kind in iter_code_examples(path):
            for seg in iter_gitee_segments(text):
                spans += 1
                tokens = tokenize(seg)
                if not tokens:
                    skipped["unparsable"] = skipped.get("unparsable", 0) + 1
                    continue
                reasons, skip = validate_command(tree, tokens)
                if skip:
                    skipped[skip] = skipped.get(skip, 0) + 1
                    continue
                checked += 1
                checked_by_kind[kind] = checked_by_kind.get(kind, 0) + 1
                for reason in reasons:
                    findings.append(
                        {
                            "file": rel,
                            "line": lineno,
                            "kind": kind,
                            "snippet": seg.strip()[:160],
                            "reason": reason,
                        }
                    )

    print(f"scanned {len(files)} markdown files, {spans} gitee invocations")
    # Counts are computed from the docs on every run (fenced blocks vs inline
    # code are counted separately so 468 vs 470-style differences are explainable).
    print(f"validated {checked} runnable examples "
          f"(fenced={checked_by_kind.get('fenced', 0)}, inline={checked_by_kind.get('inline', 0)}); "
          f"skipped {sum(skipped.values())} ({skipped})")
    if findings:
        print(f"\n{len(findings)} problem(s):")
        for f in findings[: args.limit]:
            print(f"  {f['file']}:{f['line']}: {f['reason']}")
            print(f"      {f['snippet']}")
        if args.verbose:
            for f in findings:
                print(f"    [{f['kind']}] {f['file']}:{f['line']} :: {f['snippet']}")
        if args.json:
            print(json.dumps(findings, indent=2))
        return 1
    print("all documented gitee examples are valid")
    return 0


if __name__ == "__main__":
    sys.exit(main())
