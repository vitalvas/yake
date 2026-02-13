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

	if err := checkTestFileNaming(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if err := checkCoverage(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(allErrors, "\n"))
	}

	return nil
}

func checkTestFileNaming() error {
	log.Println("Checking test file naming conventions...")

	var violations []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			switch info.Name() {
			case "vendor", ".git", "test", "tests", "examples":
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".pb.go") {
			return nil
		}

		isTestFile := strings.HasSuffix(path, "_test.go")

		if isTestFile {
			fileViolations := validateTestFileName(path)
			violations = append(violations, fileViolations...)
		} else {
			if hasTestingImport(path) {
				violations = append(violations,
					fmt.Sprintf("  - %s: file imports 'testing' but is not named '{origin}_test.go' or '{origin}_e2e_test.go'", path))
			}

			if !hasSkipDirective(path) && hasSignificantFunctions(path) {
				fileViolations := validateSourceFile(path)
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

func validateTestFileName(testPath string) []string {
	if !hasFunctions(testPath) {
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

	if !hasTestingImport(testPath) {
		violations = append(violations, fmt.Sprintf("  - %s: missing 'testing' package import", testPath))
	}

	return violations
}

func validateSourceFile(sourcePath string) []string {
	if hasSkipDirective(sourcePath) {
		return nil
	}

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

const skipDirective = "//yake:skip-test"

func hasSkipDirective(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "//") {
			if strings.HasPrefix(line, skipDirective) {
				return true
			}

			continue
		}

		if strings.HasPrefix(line, "package ") {
			break
		}
	}

	return false
}

func hasTestingImport(filePath string) bool {
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

func hasFunctions(filePath string) bool {
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

const minFunctionLines = 5
const maxUncoveredFunctionLines = 25

func hasSignificantFunctions(filePath string) bool {
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
			lines := endLine - startLine - 1

			if lines > minFunctionLines {
				return true
			}
		}
	}

	return false
}

type funcInfo struct {
	Name      string
	Lines     int
	StartLine int
	EndLine   int
}

type coverBlock struct {
	StartLine int
	EndLine   int
	Count     int
}

func largeFunctions(filePath string) []funcInfo {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil
	}

	var result []funcInfo

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		startLine := fset.Position(fn.Body.Lbrace).Line
		endLine := fset.Position(fn.Body.Rbrace).Line
		lines := endLine - startLine - 1

		if lines <= maxUncoveredFunctionLines {
			continue
		}

		name := fn.Name.Name

		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			recv := fn.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}

			if ident, ok := recv.(*ast.Ident); ok {
				name = ident.Name + "_" + fn.Name.Name
			}
		}

		result = append(result, funcInfo{
			Name:      name,
			Lines:     lines,
			StartLine: startLine,
			EndLine:   endLine,
		})
	}

	return result
}

func parseModulePath(goModContent string) string {
	scanner := bufio.NewScanner(strings.NewReader(goModContent))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}

	return ""
}

func parseCoverProfile(data, modulePath string) map[string][]coverBlock {
	result := make(map[string][]coverBlock)
	lineRegex := regexp.MustCompile(`^(.+):(\d+)\.\d+,(\d+)\.\d+\s+\d+\s+(\d+)$`)

	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "mode:") {
			continue
		}

		matches := lineRegex.FindStringSubmatch(line)
		if len(matches) != 5 {
			continue
		}

		filePath := matches[1]

		if modulePath != "" && strings.HasPrefix(filePath, modulePath+"/") {
			filePath = strings.TrimPrefix(filePath, modulePath+"/")
		}

		startLine, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}

		endLine, err := strconv.Atoi(matches[3])
		if err != nil {
			continue
		}

		count, err := strconv.Atoi(matches[4])
		if err != nil {
			continue
		}

		result[filePath] = append(result[filePath], coverBlock{
			StartLine: startLine,
			EndLine:   endLine,
			Count:     count,
		})
	}

	return result
}

func isFuncCovered(blocks []coverBlock, fn funcInfo) bool {
	for _, b := range blocks {
		if b.Count > 0 && b.StartLine >= fn.StartLine && b.StartLine <= fn.EndLine {
			return true
		}
	}

	return false
}

func findUncoveredLargeFunctions(profilePath string) ([]string, error) {
	goModData, err := os.ReadFile("go.mod")
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}

	modulePath := parseModulePath(string(goModData))

	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read coverage profile: %w", err)
	}

	coverageMap := parseCoverProfile(string(profileData), modulePath)

	var violations []string

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			switch info.Name() {
			case "vendor", ".git", "test", "tests", "examples":
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".pb.go") {
			return nil
		}

		if hasSkipDirective(path) {
			return nil
		}

		funcs := largeFunctions(path)
		if len(funcs) == 0 {
			return nil
		}

		blocks := coverageMap[path]

		for _, fn := range funcs {
			if !isFuncCovered(blocks, fn) {
				violations = append(violations,
					fmt.Sprintf("  - %s: function '%s' (%d lines) has no test coverage", path, fn.Name, fn.Lines))
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return violations, nil
}

func checkCoverage() error {
	log.Println("Checking code coverage (minimum 80% per package)...")

	tmpFile, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpFile.Close()

	defer os.Remove(tmpFile.Name())

	cmd := exec.Command("go", "test", "-coverprofile="+tmpFile.Name(), "./...")

	var stdout bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run coverage check: %w", err)
	}

	violations, err := parseCoverageOutput(stdout.String())
	if err != nil {
		return err
	}

	funcViolations, err := findUncoveredLargeFunctions(tmpFile.Name())
	if err != nil {
		return err
	}

	violations = append(violations, funcViolations...)

	if len(violations) > 0 {
		return fmt.Errorf("coverage violations (minimum %.0f%%):\n%s",
			MinCoveragePercent, strings.Join(violations, "\n"))
	}

	return nil
}

func parseCoverageOutput(output string) ([]string, error) {
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
