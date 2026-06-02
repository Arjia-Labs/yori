# Contributing to yori

Thanks for your interest in improving yori! This is a small, focused Go CLI —
contributions that keep it sharp and Unix-friendly are very welcome.

## Development

Requires Go (see the version in [`go.mod`](go.mod)).

```bash
git clone https://github.com/arjia-labs/yori
cd yori
go build -o yori .        # build the binary
go test ./...             # run the test suite
go vet ./...              # static checks
gofmt -l .                # should print nothing
```

Try your build locally without touching your real store by pointing
`YORI_HOME` at a scratch directory:

```bash
export YORI_HOME=/tmp/yori-scratch
./yori init && ./yori ls
```

## Project layout

```
main.go              # entrypoint
cmd/                 # cobra commands (one file per command)
internal/config/     # store path discovery (~/.yori, ./.yori)
internal/store/      # artifact load/save + layered resolution
internal/render/     # Liquid rendering, partials, slot inheritance
internal/registry/   # git-as-registry install/push
internal/ident/      # name validation
```

## Pull requests

- Keep changes focused; one logical change per PR.
- Add or update tests for behavior changes — every package has table-driven
  tests to follow as examples.
- Run `gofmt`, `go vet`, and `go test ./...` before pushing. CI runs all three
  (plus `-race`).
- Use clear, [Conventional Commits](https://www.conventionalcommits.org/)-style
  messages (e.g. `feat: …`, `fix: …`, `docs: …`) with a short body explaining
  the *why*.

## Reporting bugs & ideas

Open an issue using the templates. For security issues, see
[SECURITY.md](SECURITY.md) — please don't file those publicly.
