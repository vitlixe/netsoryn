# Contributing

## Requirements

- Go 1.22+
- `golangci-lint` for linting (`brew install golangci-lint` or see [golangci-lint.run](https://golangci-lint.run/welcome/install/))

## Local setup

```bash
git clone https://github.com/vitlixe/netsoryn
cd netsoryn
make build   # builds to dist/netsoryn
make run     # build + launch TUI
make test    # run tests with race detector
make lint    # run golangci-lint
```

## Development builds

**Quick dev build** — no ldflags, fast iteration:

```bash
go build -o /tmp/netsoryn-dev ./cmd/netsoryn
/tmp/netsoryn-dev
```

Because no `-ldflags` are passed, `netsoryn version` reports `dev`. This is expected for local UI testing.

**Build with git-derived version** (mirrors what CI produces):

```bash
make clean
make build
./dist/netsoryn version
./dist/netsoryn
```

`make build` passes `-ldflags` with the output of `git describe --tags --always --dirty`. If HEAD is exactly on a tag and the working tree is clean, you get `v0.1.0`. Uncommitted changes append `-dirty`, e.g. `v0.1.0-dirty`.

**Running tests:**

```bash
make test
# or explicitly:
go test ./... -race -timeout 60s
```

## Making changes

1. Fork the repo and create a branch from `main`.
2. Add or update tests where applicable.
3. Run `make test` and `make lint` — both must pass.
4. Open a PR with a clear description of what and why.

## Adding a collector

1. Implement `collectors.Collector` in `internal/collectors/`.
2. Add a view in `internal/ui/views/` with a matching `tea.Model`.
3. Register the view in `internal/ui/model.go`.

## Code style

- Standard `gofmt` formatting (enforced by lint).
- No comments that restate what the code already says.
- Keep functions small and focused.
