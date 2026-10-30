package policy

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.vitalvas.com/yake/internal/config"
)

type rustPolicyCheck struct {
	section any
	run     func() error
}

func RunRustChecks() error {
	log.Println("Running Rust policy checks...")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var allErrors []string

	for _, policyCheck := range rustPolicyChecks(cfg) {
		if !enabled(policyCheck.section) {
			continue
		}

		if err := policyCheck.run(); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(allErrors, "\n"))
	}

	return nil
}

func rustPolicyChecks(cfg *config.Config) []rustPolicyCheck {
	return []rustPolicyCheck{
		{
			section: cfg.RustPolicy.ASCIIOnly,
			run:     checkRustASCIIOnly,
		},
		{
			section: cfg.RustPolicy.NoTomlComment,
			run:     checkCargoTomlNoComment,
		},
	}
}

func checkRustASCIIOnly() error {
	log.Println("Checking Rust source for non-ASCII characters...")

	var violations []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if skipPolicyDir(info.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".rs") {
			return nil
		}

		fileViolations := findNonASCIIChars(path)
		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("ASCII-only violations (use only ASCII characters in Rust source):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func checkCargoTomlNoComment() error {
	log.Println("Checking Cargo.toml files for comments...")

	var violations []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if skipPolicyDir(info.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if info.Name() != "Cargo.toml" {
			return nil
		}

		fileViolations, err := findTomlComments(path)
		if err != nil {
			return err
		}

		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("comment violations in Cargo.toml (comments are not allowed):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func findTomlComments(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	var violations []string

	scanner := bufio.NewScanner(file)
	line := 0

	for scanner.Scan() {
		line++

		if column := tomlCommentColumn(scanner.Text()); column > 0 {
			violations = append(violations,
				fmt.Sprintf("  - %s:%d:%d: comment not allowed", path, line, column))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	return violations, nil
}

// tomlCommentColumn returns the 1-based column of a comment '#' outside any
// string on the line, or 0 when the line has no comment. It tracks basic
// ("...") and literal ('...') TOML strings so a '#' inside a value is ignored.
func tomlCommentColumn(text string) int {
	var quote byte

	for i := 0; i < len(text); i++ {
		c := text[i]

		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else if quote == '"' && c == '\\' {
				i++
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return i + 1
		}
	}

	return 0
}
