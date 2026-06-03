<p align="center">
  <img src="assets/header.jpg" alt="yori — the home for everything you tell your AI" width="100%">
</p>

<h1 align="center">yori</h1>

<p align="center">
  <em>📚 The home for everything you tell your AI.</em><br>
  <em>Named after Tron's <strong>Yori</strong>, who ran the I/O Tower — the gateway where a User's words reached their program.</em>
</p>

<p align="center">
  <a href="https://github.com/arjia-labs/yori/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/arjia-labs/yori/ci.yml?branch=main&style=flat-square&logo=github&label=build" alt="Build status"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="MIT licensed"></a>
  <img src="https://img.shields.io/badge/go-1.26%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.26+">
  <img src="https://img.shields.io/badge/CGo-free-success?style=flat-square" alt="No CGo">
  <img src="https://img.shields.io/badge/unix-pipe--friendly-333?style=flat-square" alt="Pipe friendly">
</p>

<p align="center">
  <a href="#-quickstart"><strong>Quickstart</strong></a> ·
  <a href="#-concepts"><strong>Concepts</strong></a> ·
  <a href="#-templating"><strong>Templating</strong></a> ·
  <a href="#-registry"><strong>Registry</strong></a> ·
  <a href="#-command-reference"><strong>Commands</strong></a> ·
  <a href="#-design-notes"><strong>Design</strong></a>
</p>

---

## 🤔 Why?

Right now your prompts live everywhere and nowhere — half-buried in code, pasted into a dozen chat windows, screenshotted in a Slack thread, slightly reworded by every person on the team. Nobody's sure which version is the good one.

`yori` gives all those words a single place to live: a local, file-based library of reusable AI building blocks — prompts, agents, slash-commands, skills — that you can list, edit, compose, and render into ready-to-pipe text.

It's a **pure renderer**: text in, text out. `yori` never calls a model. You own the model invocation — pipe the rendered prompt into `claude`, `llm`, or anything else.

```bash
cat bug.log | yori run triage --tone=blunt | claude
```

## ✨ Highlights

| | |
|---|---|
| 📄 **Plain files** | Every artifact is a markdown file with YAML frontmatter. Greppable, `$EDITOR`-friendly, diffs cleanly, versioned with your own git. |
| 🧩 **Unified artifacts** | One home for prompts, **agents**, **slash-commands**, and **skills** — same treatment, organized by type. |
| 🪢 **Composition** | Liquid templating: `{{ variables }}`, `{% include %}` partials, `{% if %}`/`{% for %}`, and **template inheritance** via slots. |
| 🚰 **Pipe-first** | Reads stdin, writes stdout. `{{ input }}` captures piped text (or it's appended). Drops into any Unix pipeline. |
| 🗂️ **Layered store** | A project `./.yori` shadows your global `~/.yori`, which is backed by installed packages — like a search path for prompts. |
| 📦 **Git-as-registry** | `yori install <git-url>` to pull a team's shelf; `yori push` to publish yours. No server — git is the transport. |
| 🔌 **Deploy to agents** | `yori sync` renders skills, commands, and subagents into the dirs Claude Code, Codex, and Cursor discover them from. Compose once, deploy everywhere. |
| 🧪 **Eval via promptfoo** | `yori export promptfoo` turns a composed artifact + its cases into a promptfooconfig.yaml. yori manages and ships; promptfoo grades. |
| 🚫 **No model, no network telemetry** | Pure text transform. The only network use is git, when you ask for it. |

## 📦 Install

```bash
go install github.com/arjia-labs/yori@latest
```

Add `$HOME/go/bin` to your `PATH`, then verify with `yori --help`.

Or from a clone:

```bash
go build -o yori . && ./yori --help
```

## 🚀 Quickstart

```bash
mkdir my-project && cd my-project
yori init                                  # 📂 creates ./.yori/store

yori add triage                            # ✍️  opens $EDITOR with a scaffold
# ... write a prompt that uses {{ tone }} and {{ input }} ...

yori ls                                     # 📋 list everything
yori show triage                            # 🔎 metadata + declared variables

echo "NullPointer at line 42" | yori run triage --tone=blunt   # 🎬 render to stdout
echo "NullPointer at line 42" | yori run triage --tone=blunt | claude   # 🤝 pipe to your model
```

A `triage.md` looks like this:

```markdown
---
name: triage
description: Triage a bug from a log
tags: [debug, ops]
vars:
  tone:
    default: neutral
    description: voice of the response
---
{% include 'house-style' %}

Analyze this log as a {{ tone }} engineer:

{{ input }}
```

## 🧠 Concepts

### Artifacts and types

An **artifact** is one markdown file: YAML frontmatter (`name`, `description`, `tags`, `model`, `extends`, `vars`) plus a Liquid body. There are four types, each in its own subfolder of a store:

| type | folder | what it is |
|---|---|---|
| `prompt` *(default)* | `store/` | a reusable prompt |
| `agent` | `store/agents/` | a system prompt + role definition |
| `command` | `store/commands/` | a slash-command body |
| `skill` | `store/skills/` | a skill description |

Every command takes `--type`/`-t` (default `prompt`). `yori ls` shows all types by default with a `TYPE` column; `yori ls --type agent` filters.

```bash
yori add pr-bot --type agent
yori run pr-bot -t agent --file=diff.patch
yori ls --type command
```

### The layered store

Artifacts resolve through layers, highest priority first:

1. **Project** — `./.yori/store` (discovered by walking up from the working directory, like `.git`)
2. **Global** — `~/.yori/store`
3. **Installed packages** — `~/.yori/pkg/<name>` (read-only)

A project artifact shadows a same-named global one, which shadows a package one. Address a package artifact explicitly as `<pkg>/<name>`. Most write commands take `--global` to target `~/.yori` instead of the project.

## 🪢 Templating

Bodies are rendered with [Liquid](https://shopify.github.io/liquid/) — text-native, no HTML escaping.

**Variables** are filled at call time, in this precedence (high → low):

| source | example |
|---|---|
| CLI flag | `--tone=blunt` |
| `--set` (for names that clash with reserved flags) | `--set file=README` |
| `@file` injection (any value) | `--notes=@notes.md` |
| piped stdin → `input` | `cat x.log \| yori run triage` |
| frontmatter `vars.<name>.default` | `tone: { default: neutral }` |
| *(blank)* | undefined renders empty |

**Stdin** fills `{{ input }}`. If the template doesn't reference it, the piped text is appended to the end instead of dropped. `--file=PATH` is an alternative source for `{{ input }}`.

**Partials** — share a block across prompts:

```liquid
{% include 'house-style' %}     {# loads store/partials/house-style.md #}
```

**Logic** — Liquid conditionals and loops work:

```liquid
{% if examples %}Examples:
{% for e in examples %}- {{ e }}
{% endfor %}{% endif %}
```

**Template inheritance (slots)** — a base defines overridable regions; a child `extends` it and fills them:

```markdown
# base.md
You are an assistant.
{% slot "guidelines" %}Be concise.{% endslot %}

Task: {{ input }}
```

```markdown
# verbose.md
---
name: verbose
extends: base
---
{% fill "guidelines" %}Be verbose and cite your sources.{% endfill %}
```

Rendering `verbose` pours the fill into the base's slot; unfilled slots fall back to their default. Inheritance chains and is cycle-checked.

## 📦 Registry

`yori` treats a git repo whose root is a store as a shareable **package** — like Go modules for prompts.

```bash
# Pull a team's shelf (shallow-cloned to ~/.yori/pkg/<name>, pinned to a commit)
yori install https://github.com/acme/prompts --name acme
yori pkg ls
yori run acme/review                      # address it explicitly
yori update acme                          # fast-forward + re-pin
yori uninstall acme

# Publish your own global store
yori push --remote git@github.com:me/prompts.git -m "initial set"
yori push                                 # subsequent pushes need no flags
```

Installed packages are read-only layers, so `yori run review` falls through to a package if nothing local matches. Transport is plain `git` shelled out — no server to run, nothing extra to install.

### Items, not just packages (`.yori.json`)

`yori registry build` generates a `.yori.json` manifest — yori **infers** each item's files and dependencies from the composition graph (no hand-authoring). Then anyone can discover and install *individual* items, with their dependency closure, as editable source:

```bash
# Publish (one command: build .yori.json + commit + push the global store)
yori publish --remote github.com/me/prompts

# Consume — alias once, then a single fetch + vendor + deploy
yori registry add acme github.com/me/prompts
yori view acme                          # browse items (no clone for public GitHub)
yori install acme pr-summary --sync     # vendor the item + its deps, then deploy to your agent
```

Unlike a whole-package install (a read-only layer), **per-item install** copies the item, the base it extends, and the partials it includes into your store as source you own. The manifest is auto-generated from the composition graph and agent-readable — an agent can read it to know exactly what a registry offers and how to compose it. Bare URLs work for public *and* private repos (https with an ssh fallback).

### Install what your project needs

An item can declare a `when:` condition; `yori install --auto` reads your project's dependency manifests (`package.json`, `go.mod`, `pyproject.toml`, `Cargo.toml`, …) and installs only what applies. shadcn installs what you ask; **yori installs what your stack implies.**

```markdown
---
name: nextjs-helper
when: { deps: [next] }      # only relevant when `next` is a dependency
tags: [frontend]
---
```

```bash
yori detect                          # see the stack yori detects
yori install acme --auto             # everything applicable to this project
yori install acme --tag frontend     # everything tagged frontend
yori install acme --auto --tag frontend   # both: frontend items that fit the stack
```

So a single registry serves a Next.js app and a NestJS service differently — each gets the items its dependencies call for. `yori sync --tag <t>` likewise deploys only the artifacts you've tagged.

## 🔌 Use it with your agent (`yori sync`)

A skill or command only helps if your coding agent can *find* it. `yori sync` materializes your skills and commands into the directories agents discover them from — rendering templates (vars, includes, slots) on the way, so you compose once and deploy everywhere.

```bash
yori sync                          # render into ./.claude (Claude Code, default)
yori sync -a codex -a cursor       # target specific agents (repeatable)
yori sync -a '*'                   # every supported agent
yori sync --global                 # into the agent's global dir (personal)
yori sync --set tone=blunt         # override template variables at deploy time
yori sync --link                   # symlink static artifacts (live editing)
yori unsync -a '*'                 # remove everything sync placed
```

Supported agents and where each type lands (project scope shown; `--global` uses the personal dir):

| | skill | command | agent / subagent |
|---|---|---|---|
| **claude-code** | `.claude/skills/<n>/SKILL.md` | `.claude/commands/<n>.md` | `.claude/agents/<n>.md` |
| **codex** | `.agents/skills/<n>/SKILL.md` | `~/.codex/prompts/<n>.md` *(global only)* | — |
| **cursor** | — | `.cursor/commands/<n>.md` | — |

Skills carry their bundle support files; a command's `{{ input }}` becomes the agent's argument token (`$ARGUMENTS`); a yori `agent` becomes a Claude subagent with `name`/`description`/`model` frontmatter. Combinations an agent doesn't support are skipped with a note, never an error.

**Frontmatter passes through.** Any agent-specific frontmatter you author — `allowed-tools`, `argument-hint`, `agent`, `context`, `tools` — is preserved on deploy, and runtime syntax (`!`cmd``, `$ARGUMENTS`, `@file`) is never touched. So you can compose a dynamic command from shared partials *and* let the agent execute it:

```markdown
---
name: pr-summary
description: Summarize a pull request
allowed-tools: Bash(gh *)
---
{% include 'house-style' %}        {# yori composes the static scaffolding #}
PR diff: !`gh pr diff`             {# the agent runs this at invocation #}
Summarize for {{ input }}.         {# → $ARGUMENTS #}
```

yori records what it wrote, so a re-sync **prunes** artifacts you've removed and refuses to clobber files it didn't create (use `--force` to override). This is the piece a plain installer can't do: the deployed skill is a *rendered, parameterized* copy of your source, not a raw file.

**Make it reproducible.** `yori sync --save` records the chosen artifacts to a committed `.yori/sync.yaml`. Then a teammate clones the repo and runs a bare `yori sync` to hydrate the project's whole agent setup in one command:

```yaml
# .yori/sync.yaml
agents: [claude-code]
artifacts: [researcher, triage]
```

## 🧪 Evaluate with promptfoo (`yori export`)

yori is the source of truth for your prompts; for model-graded evaluation it hands off to [promptfoo](https://promptfoo.dev) rather than reinventing it. `yori export promptfoo <name>` resolves an artifact's composition (includes, slots) but leaves variables as `{{ placeholders }}`, and emits a ready-to-run config:

```bash
yori export promptfoo review > promptfooconfig.yaml
promptfoo eval
```

Test cases live next to the artifact in `<name>.cases.yaml` (or `cases.yaml` in a skill bundle) — a plain list of promptfoo test objects, so authoring yori cases *is* authoring promptfoo tests:

```yaml
# review.cases.yaml
- description: flags a real bug
  vars: { lang: go, input: "func f() int { }" }
  assert:
    - { type: contains, value: "return" }
    - { type: llm-rubric, value: "explains the root cause" }
```

The provider comes from `--provider` or the artifact's `model:` hint. yori *manages and ships* your prompts; promptfoo *grades* them.

## 📋 Command reference

| command | what it does |
|---|---|
| `yori init` | create a project store (`./.yori/store`) |
| `yori add <name>` | scaffold a new artifact and open `$EDITOR` |
| `yori edit <name>` | open an existing artifact in `$EDITOR` |
| `yori get <name>` | print the **raw** body (no rendering) |
| `yori run <name>` | **render** — fill variables, inject stdin, print to stdout |
| `yori show <name>` | print metadata (type, layer, path, tags, vars) |
| `yori ls` | list artifacts (all types; `--type`, `--tag`, `--global`) |
| `yori which <name>` | print the resolved file path |
| `yori deps <name>` | what an artifact composes from (extends + transitive includes) |
| `yori affected <name>` | which artifacts include/extend a partial or base (blast radius) |
| `yori rm <name>` | delete an artifact |
| `yori install <reg> [items]` | install a package, or vendor items (`--auto`, `--tag`, `--all`, `--sync`, `--global`) |
| `yori detect` | print the project stack `--auto` matches `when:` conditions against |
| `yori publish` | build the manifest + commit + push the global store (`--remote`, `-m`) |
| `yori registry build` | generate a `.yori.json` manifest from the store (`--out`, `--global`) |
| `yori registry add/ls/rm` | manage registry aliases (use a short name with `install`/`view`) |
| `yori view <reg> [item]` | browse a registry's items from its manifest, no clone (`--all`) |
| `yori pkg ls` | list installed packages |
| `yori update [name]` | pull + re-pin installed packages |
| `yori uninstall <name>` | remove an installed package |
| `yori push` | publish the global store to a git remote (`--remote`, `-m`) |
| `yori sync [names]` | render skills + commands into an agent's dirs (`--agent`, `--global`, `--link`, `--set`, `--force`, `--save`) |
| `yori unsync` | remove what `yori sync` placed (`--agent`, `--global`) |
| `yori export promptfoo <name>` | generate a promptfooconfig.yaml for evaluation (`--provider`, `--type`) |

Shared flags: `--type`/`-t` selects the artifact type; `--global` targets `~/.yori` instead of the project.

## 🧭 Design notes

- **Pure renderer, by choice.** `yori` is a sharp Unix filter, not a model client. No API keys, no SDKs, no opinions about which model you use — it just produces the text and gets out of the way.
- **Files are the source of truth.** No database, no lock-in. The store is a directory of markdown you can read, grep, edit by hand, and commit. Versioning is your own git.
- **Git is the network.** Sharing reuses infrastructure everyone already has. A package is just a repo; publishing is `git push`; installing is `git clone`.
- **Forgiving rendering.** Undefined variables render blank and missing values fall back to frontmatter defaults — a half-filled prompt never hard-fails mid-pipeline.

## 🗺️ Roadmap

Out of scope today, but the file layout leaves room for: lightweight evals (regression-test a prompt across versions), a render-once → symlink-many sync mode for multi-agent setups, more agent targets, and richer namespacing.
