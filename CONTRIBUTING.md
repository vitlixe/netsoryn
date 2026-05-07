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
