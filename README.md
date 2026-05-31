<h1 align="center">yori</h1>

<p align="center">
  <em>📚 The home for everything you tell your AI.</em><br>
  <em>Named after Tron's <strong>Yori</strong>, who ran the I/O Tower — the gateway where a User's words reached their program.</em>
</p>

<p align="center">
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
| 🚫 **No model, no network telemetry** | Pure text transform. The only network use is git, when you ask for it. |

## 📦 Install

```bash
go install github.com/rovak/yori@latest
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
| `yori rm <name>` | delete an artifact |
| `yori install <git-url>` | install a package from git (`--name`) |
| `yori pkg ls` | list installed packages |
| `yori update [name]` | pull + re-pin installed packages |
| `yori uninstall <name>` | remove an installed package |
| `yori push` | publish the global store to a git remote (`--remote`, `-m`) |

Shared flags: `--type`/`-t` selects the artifact type; `--global` targets `~/.yori` instead of the project.

## 🧭 Design notes

- **Pure renderer, by choice.** `yori` is a sharp Unix filter, not a model client. No API keys, no SDKs, no opinions about which model you use — it just produces the text and gets out of the way.
- **Files are the source of truth.** No database, no lock-in. The store is a directory of markdown you can read, grep, edit by hand, and commit. Versioning is your own git.
- **Git is the network.** Sharing reuses infrastructure everyone already has. A package is just a repo; publishing is `git push`; installing is `git clone`.
- **Forgiving rendering.** Undefined variables render blank and missing values fall back to frontmatter defaults — a half-filled prompt never hard-fails mid-pipeline.

## 🗺️ Roadmap

Out of scope today, but the file layout leaves room for: lightweight evals (regression-test a prompt across versions), `.yori` project dependency hydration / lockfiles, multi-file skill bundles, and richer namespacing.
