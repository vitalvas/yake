package policy

import (
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
