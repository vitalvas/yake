package policy

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const MinCoveragePercent = 80.0

func RunGolangChecks() error {
	log.Println("Running Go policy checks...")

	var allErrors []string

	if err := CheckTestFileNaming(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if err := CheckCoverage(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(allErrors, "\n"))
	}

	return nil
}

func CheckTestFileNaming() error {
	log.Println("Checking test file naming conventions...")

	var violations []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			switch info.Name() {
			case "vendor", ".git", "test", "tests":
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		isTestFile := strings.HasSuffix(path, "_test.go")

		if isTestFile {
			fileViolations := ValidateTestFileName(path)
			violations = append(violations, fileViolations...)
		} else {
			if HasTestingImport(path) {
				violations = append(violations,
					fmt.Sprintf("  - %s: file imports 'testing' but is not named '{origin}_test.go' or '{origin}_e2e_test.go'", path))
			}

			if HasSignificantFunctions(path) {
				fileViolations := ValidateSourceFile(path)
				violations = append(violations, fileViolations...)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("test file naming violations:\n%s", strings.Join(violations, "\n"))
	}

	return nil
}

func ValidateTestFileName(testPath string) []string {
	if !HasFunctions(testPath) {
		return nil
	}

	var violations []string

	filename := filepath.Base(testPath)
	dir := filepath.Dir(testPath)

	isE2ETest := strings.HasSuffix(filename, "_e2e_test.go")

	var expectedSourceFile string

	if isE2ETest {
		baseName := strings.TrimSuffix(filename, "_e2e_test.go")
		expectedSourceFile = baseName + ".go"
	} else {
		baseName := strings.TrimSuffix(filename, "_test.go")
		expectedSourceFile = baseName + ".go"

		invalidPatterns := []string{
			"_unit_test.go",
			"_bench_test.go",
			"_integration_test.go",
		}

		for _, pattern := range invalidPatterns {
			if strings.HasSuffix(filename, pattern) {
				violations = append(violations, fmt.Sprintf("  - %s: invalid naming pattern '%s', use '{origin}_test.go' or '{origin}_e2e_test.go'",
					testPath, pattern))
			}
		}
	}

	sourcePath := filepath.Join(dir, expectedSourceFile)

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		violations = append(violations, fmt.Sprintf("  - %s: missing source file '%s'", testPath, sourcePath))
	}

	if !HasTestingImport(testPath) {
		violations = append(violations, fmt.Sprintf("  - %s: missing 'testing' package import", testPath))
	}

	return violations
}

func ValidateSourceFile(sourcePath string) []string {
	var violations []string

	filename := filepath.Base(sourcePath)
	dir := filepath.Dir(sourcePath)

	baseName := strings.TrimSuffix(filename, ".go")
	testFile := baseName + "_test.go"
	testPath := filepath.Join(dir, testFile)

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		violations = append(violations,
			fmt.Sprintf("  - %s: missing test file '%s'", sourcePath, testPath))
	}

	return violations
}

func HasTestingImport(filePath string) bool {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return false
	}

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if importPath == "testing" {
			return true
		}
	}

	return false
}

func HasFunctions(filePath string) bool {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return false
	}

	for _, decl := range node.Decls {
		if _, ok := decl.(*ast.FuncDecl); ok {
			return true
		}
	}

	return false
}

const minFunctionLines = 3

func HasSignificantFunctions(filePath string) bool {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return false
	}

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Body == nil {
				continue
			}

			startLine := fset.Position(fn.Body.Lbrace).Line
			endLine := fset.Position(fn.Body.Rbrace).Line
			lines := endLine - startLine + 1

			if lines > minFunctionLines {
				return true
			}
		}
	}

	return false
}

func CheckCoverage() error {
	log.Println("Checking code coverage (minimum 80% per package)...")

	cmd := exec.Command("go", "test", "-cover", "./...")

	var stdout bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run coverage check: %w", err)
	}

	violations, err := ParseCoverageOutput(stdout.String())
	if err != nil {
		return err
	}

	if len(violations) > 0 {
		return fmt.Errorf("coverage violations (minimum %.0f%%):\n%s",
			MinCoveragePercent, strings.Join(violations, "\n"))
	}

	return nil
}

func ParseCoverageOutput(output string) ([]string, error) {
	var violations []string

	coverageRegex := regexp.MustCompile(`ok\s+(\S+)\s+(?:[\d.]+s|\(cached\))\s+coverage:\s+([\d.]+)%`)
	noCoverageRegex := regexp.MustCompile(`\?\s+(\S+)\s+\[no test files\]`)

	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		if matches := coverageRegex.FindStringSubmatch(line); len(matches) == 3 {
			pkgName := matches[1]

			coverage, err := strconv.ParseFloat(matches[2], 64)
			if err != nil {
				continue
			}

			if coverage < MinCoveragePercent {
				violations = append(violations,
					fmt.Sprintf("  - %s: %.1f%% coverage (minimum %.0f%%)", pkgName, coverage, MinCoveragePercent))
			}
		}

		if matches := noCoverageRegex.FindStringSubmatch(line); len(matches) == 2 {
			violations = append(violations,
				fmt.Sprintf("  - %s: no test files", matches[1]))
		}
	}

	return violations, scanner.Err()
}
