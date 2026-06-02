# Security Policy

## Reporting a vulnerability

Please report security issues **privately** — do not open a public issue.

Use GitHub's [private vulnerability reporting](https://github.com/arjia-labs/yori/security/advisories/new)
("Report a vulnerability" under the repository's **Security** tab).

Please include:

- a description of the issue and its impact,
- steps to reproduce (a minimal repro is ideal), and
- affected version / commit.

We'll acknowledge your report, investigate, and coordinate a fix and
disclosure timeline with you.

## Scope

yori reads and writes files under a store directory, executes `git` for the
registry commands, and renders user-authored templates. Reports of particular
interest include:

- path traversal or writes/deletes outside the intended store or package root,
- template rendering that escapes its sandbox or executes unintended commands,
- registry install/update operating outside `~/.yori/pkg`.
