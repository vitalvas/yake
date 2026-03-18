# YaKe

Yet Another ToolKit -- a CLI tool for enforcing development standards and generating project configurations.

## Installation

```bash
GOPRIVATE=github.com/vitalvas/yake go install -v github.com/vitalvas/yake@latest
```

## Commands

### `yake tests`

Runs a comprehensive testing and quality pipeline:

- `go fmt` -- format code
- `go vet` -- static analysis
- `go mod tidy -v` -- clean dependencies
- `go test -cover ./...` -- test coverage
- `go test -race ./...` -- race detector
- `golangci-lint run` -- linting (if `.golangci.yml` exists)
- `goreleaser check` -- release config validation (if `.goreleaser.yml` exists)

### `yake policy run`

Enforces project-wide policies for Go projects:

- **Entry point layout** -- root `main.go` and `cmd/**/main.go` must not coexist; entry point files may only contain `main()` (no `init()`, no helpers, max 25 lines)
- **Package naming** -- package names must match `^[0-9a-z]{3,32}$` and match their directory name
- **Test file naming** -- test files must follow `{origin}_test.go` or `{origin}_e2e_test.go` convention
- **Code coverage** -- minimum 80% coverage per package; large functions (>25 lines) must have test coverage; packages without testable code (e.g., embed-only) are skipped

#### Skip directives

- `//yake:skip-test` before the `package` declaration skips test requirements for the entire file
- `//yake:skip-test` above a function declaration skips coverage requirements for that function

### `yake code`

Generates project configurations:

| Subcommand | Description |
|---|---|
| `defaults` | Apply default project configurations (e.g., `.golangci.yml`) |
| `linter-new --lang go` | Create a new linter configuration |
| `github-dependabot --lang go` | Generate `.github/dependabot.yml` |
| `github-release-please` | Generate Release Please workflow and config |
| `github-lang-golang` | Generate GitHub Actions CI workflow for Go |
