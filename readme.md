# YaKe

Yet Another ToolKit -- a CLI tool for enforcing development standards and generating project configurations.

## Installation

```bash
GOPRIVATE=github.com/vitalvas/yake go install -v github.com/vitalvas/yake@latest
```

## Configuration

All settings are configured via `.yake.yaml` in the project root. Every field is optional -- defaults are applied when omitted.

```yaml
tests:
  tags:                       # Go build tags applied to vet, test, and race runs
    - integration
    - e2e

policy:
  entry_points:
    enable: true              # default: true
    max_main_lines: 25        # default: 25

  package_naming:
    enable: true              # default: true
    pattern: "^[0-9a-z]{3,32}$"  # default: ^[0-9a-z]{3,32}$

  ascii_only:
    enable: true              # default: true

  string_concat:
    enable: true              # default: true

  stdlib_wrappers:
    enable: true              # default: true

  func_signature:
    enable: true              # default: true
    max_params: 5             # default: 5
    max_results: 5            # default: 5

  composite_literal:
    enable: true              # default: true
    max_single_line_fields: 5 # default: 5

  stuttering:
    enable: true              # default: true

  getter_naming:
    enable: true              # default: true

  private_exported_methods:
    enable: true              # default: true

  test_file_naming:
    enable: true              # default: true

  test_duration:
    enable: true              # default: true
    max_duration: "10s"       # default: 10s

  coverage:
    enable: true              # default: true
    min_coverage: 80.0        # default: 80.0
    max_uncovered_func_lines: 25  # default: 25
    exclude_packages:         # packages excluded from all coverage checks
      - internal/cmd
      - internal/generated
    package_overrides:        # per-package minimum coverage (overrides min_coverage)
      internal/database: 50.0
      internal/cli: 40.0
```

### Build tags

`tests.tags` lists Go build tags applied when running tests. The untagged
`go vet`, `go test -cover`, and `go test -race` run always executes. When tags are
configured, an additional tagged pass runs on top, with each tag passed as a separate
`-tags` flag, so both tagged and untagged code paths are exercised. For example,
`tags: [integration, e2e]` runs the untagged pass followed by
`go test -cover -tags=integration -tags=e2e ./...` (and the matching `vet`/`race`).

### Skip directives

- `//yake:skip-test` before the `package` declaration skips test requirements for the entire file
- `//yake:skip-test` above a function declaration skips coverage requirements for that function
