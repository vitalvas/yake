package policy

import (
	"bufio"
	"bytes"
	"encoding/json"
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
	"time"
)

const (
	MinCoveragePercent        = 80.0
	maxTestDuration           = 10 * time.Second
	minFunctionLines          = 5
	maxUncoveredFunctionLines = 25
	skipDirective             = "//yake:skip-test"
)

var packageNameRegex = regexp.MustCompile(`^[0-9a-z]{3,32}$`)

func RunGolangChecks() error {
	log.Println("Running Go policy checks...")

	var allErrors []string

	if err := checkEntryPoints(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if err := checkPackageNaming(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if err := checkStringConcat(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if err := checkStdlibWrappers(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if err := checkTestFileNaming(); err != nil {
		allErrors = append(allErrors, err.Error())
	}

	if err := checkTestDuration(); err != nil {
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

func checkEntryPoints() error {
	log.Println("Checking entry point layout (root main.go vs cmd/**/main.go)...")

	hasRootMain := false

	if _, err := os.Stat("main.go"); err == nil {
		hasRootMain = true
	}

	var cmdMains []string

	_ = filepath.Walk("cmd", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.Name() == "main.go" {
			cmdMains = append(cmdMains, path)
		}

		return nil
	})

	if hasRootMain && len(cmdMains) > 0 {
		return fmt.Errorf("entry point violation: found both root main.go and cmd/ entry points; use one layout:\n  - root main.go (single binary)\n  - cmd/*/main.go (multiple binaries)")
	}

	var mainFiles []string

	if hasRootMain {
		mainFiles = append(mainFiles, "main.go")
	}

	mainFiles = append(mainFiles, cmdMains...)

	violations := validateMainFiles(mainFiles)
	if len(violations) > 0 {
		return fmt.Errorf("entry point violations:\n%s", strings.Join(violations, "\n"))
	}

	return nil
}

func validateMainFiles(paths []string) []string {
	var violations []string

	for _, path := range paths {
		fset := token.NewFileSet()

		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			continue
		}

		hasMain := false

		for _, decl := range node.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if fn.Name.Name == "main" && fn.Recv == nil {
				hasMain = true

				if fn.Body != nil {
					lines := countCodeLines(fset, path, fn.Body.Lbrace, fn.Body.Rbrace)

					if lines > maxUncoveredFunctionLines {
						violations = append(violations,
							fmt.Sprintf("  - %s: main() is %d lines (maximum %d); move logic to internal/ or pkg/",
								path, lines, maxUncoveredFunctionLines))
					}
				}

				continue
			}

			violations = append(violations,
				fmt.Sprintf("  - %s: unexpected function '%s'; only main() is allowed in entry point files",
					path, fn.Name.Name))
		}

		if !hasMain {
			violations = append(violations,
				fmt.Sprintf("  - %s: missing main() function", path))
		}
	}

	return violations
}

func checkStringConcat() error {
	log.Println("Checking for string concatenation with '+'...")

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

		fileViolations := findStringConcatenations(path)
		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("string concatenation violations (use fmt.Sprintf or strings.Builder):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func findStringConcatenations(filePath string) []string {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var violations []string

	ast.Inspect(node, func(n ast.Node) bool {
		binExpr, ok := n.(*ast.BinaryExpr)
		if !ok || binExpr.Op != token.ADD {
			return true
		}

		if isStringLit(binExpr.X) || isStringLit(binExpr.Y) {
			pos := fset.Position(binExpr.OpPos)
			violations = append(violations,
				fmt.Sprintf("  - %s:%d: string concatenation with '+'", filePath, pos.Line))
		}

		return true
	})

	return violations
}

func isStringLit(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)

	return ok && lit.Kind == token.STRING
}

func checkStdlibWrappers() error {
	log.Println("Checking for stdlib wrapper functions...")

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

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".pb.go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileViolations := findStdlibWrappers(path)
		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("stdlib wrapper violations (do not wrap standard library functions):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func findStdlibWrappers(filePath string) []string {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	stdlibAliases := buildStdlibAliases(node)
	if len(stdlibAliases) == 0 {
		return nil
	}

	var violations []string

	if hasSkipDirective(filePath) {
		return nil
	}

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Recv != nil {
			continue
		}

		if hasFuncSkipDirective(fn) {
			continue
		}

		if len(fn.Body.List) != 1 {
			continue
		}

		retStmt, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(retStmt.Results) != 1 {
			continue
		}

		callExpr, ok := retStmt.Results[0].(*ast.CallExpr)
		if !ok {
			continue
		}

		pkgFunc := extractPkgFunc(callExpr)
		if pkgFunc == "" {
			continue
		}

		parts := strings.SplitN(pkgFunc, ".", 2)
		if len(parts) != 2 {
			continue
		}

		if _, isStdlib := stdlibAliases[parts[0]]; !isStdlib {
			continue
		}

		if !isParamForwarding(fn, callExpr) {
			continue
		}

		pos := fset.Position(fn.Pos())
		violations = append(violations,
			fmt.Sprintf("  - %s:%d: function '%s' is a wrapper around '%s'",
				filePath, pos.Line, fn.Name.Name, pkgFunc))
	}

	return violations
}

func buildStdlibAliases(node *ast.File) map[string]string {
	aliases := make(map[string]string)

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if !isStdlibImport(importPath) {
			continue
		}

		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			alias = parts[len(parts)-1]
		}

		if alias != "_" && alias != "." {
			aliases[alias] = importPath
		}
	}

	return aliases
}

func isStdlibImport(importPath string) bool {
	if !strings.Contains(importPath, ".") {
		return true
	}

	return strings.HasPrefix(importPath, "golang.org/x/")
}

func extractPkgFunc(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}

	return fmt.Sprintf("%s.%s", ident.Name, sel.Sel.Name)
}

func isParamForwarding(fn *ast.FuncDecl, call *ast.CallExpr) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return len(call.Args) == 0
	}

	paramNames := make(map[string]bool)
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			paramNames[name.Name] = true
		}
	}

	if len(paramNames) == 0 {
		return false
	}

	usedParams := 0
	for _, arg := range call.Args {
		ident, ok := arg.(*ast.Ident)
		if !ok {
			continue
		}

		if paramNames[ident.Name] {
			usedParams++
		}
	}

	return usedParams == len(paramNames)
}

func checkPackageNaming() error {
	log.Println("Checking package naming conventions...")

	absRoot, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve root directory: %w", err)
	}

	rootDirName := filepath.Base(absRoot)

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

		pkgName := extractPackageName(path)
		if pkgName == "" || pkgName == "main" {
			return nil
		}

		if !packageNameRegex.MatchString(pkgName) {
			violations = append(violations,
				fmt.Sprintf("  - %s: package name '%s' does not match '^[0-9a-z]{3,32}$'", path, pkgName))
		}

		dirName := filepath.Base(filepath.Dir(path))
		if dirName == "." {
			dirName = rootDirName
		}

		if dirName != pkgName {
			violations = append(violations,
				fmt.Sprintf("  - %s: package name '%s' does not match directory name '%s'", path, pkgName, dirName))
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("package naming violations:\n%s", strings.Join(violations, "\n"))
	}

	return nil
}

func extractPackageName(filePath string) string {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.PackageClauseOnly)
	if err != nil {
		return ""
	}

	return node.Name.Name
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
		expectedSourceFile = fmt.Sprintf("%s.go", baseName)
	} else {
		baseName := strings.TrimSuffix(filename, "_test.go")
		expectedSourceFile = fmt.Sprintf("%s.go", baseName)

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

	if !isStandardTestFile(filename) {
		sourcePath := filepath.Join(dir, expectedSourceFile)

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			violations = append(violations, fmt.Sprintf("  - %s: missing source file '%s'", testPath, sourcePath))
		}
	}

	if !isStandardTestFile(filename) && !hasTestingImport(testPath) {
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
	testFile := fmt.Sprintf("%s_test.go", baseName)
	testPath := filepath.Join(dir, testFile)

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		violations = append(violations,
			fmt.Sprintf("  - %s: missing test file '%s'", sourcePath, testPath))
	}

	return violations
}

var standardTestFiles = map[string]bool{
	"example_test.go":  true,
	"external_test.go": true,
}

func isStandardTestFile(filename string) bool {
	return standardTestFiles[filename]
}

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

func hasFuncSkipDirective(fn *ast.FuncDecl) bool {
	if fn.Doc == nil {
		return false
	}

	for _, comment := range fn.Doc.List {
		if strings.HasPrefix(strings.TrimSpace(comment.Text), skipDirective) {
			return true
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

func countCodeLines(fset *token.FileSet, filePath string, lbrace, rbrace token.Pos) int {
	startLine := fset.Position(lbrace).Line
	endLine := fset.Position(rbrace).Line

	data, err := os.ReadFile(filePath)
	if err != nil {
		return endLine - startLine - 1
	}

	lines := strings.Split(string(data), "\n")

	count := 0
	for i := startLine; i < endLine-1; i++ {
		if i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			count++
		}
	}

	return count
}

func hasSignificantFunctions(filePath string) bool {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return false
	}

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Body == nil || hasFuncSkipDirective(fn) {
				continue
			}

			lines := countCodeLines(fset, filePath, fn.Body.Lbrace, fn.Body.Rbrace)
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

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var result []funcInfo

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || hasFuncSkipDirective(fn) {
			continue
		}

		startLine := fset.Position(fn.Body.Lbrace).Line
		endLine := fset.Position(fn.Body.Rbrace).Line
		lines := countCodeLines(fset, filePath, fn.Body.Lbrace, fn.Body.Rbrace)

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
				name = fmt.Sprintf("%s_%s", ident.Name, fn.Name.Name)
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

		prefix := fmt.Sprintf("%s/", modulePath)
		if modulePath != "" && strings.HasPrefix(filePath, prefix) {
			filePath = strings.TrimPrefix(filePath, prefix)
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

func checkTestDuration() error {
	log.Printf("Checking test duration (maximum %s per package)...", maxTestDuration)

	cmd := exec.Command("go", "test", "-json", fmt.Sprintf("-timeout=%s", maxTestDuration), "./...")
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		if len(out) > 0 {
			violations := parseTestDurationOutput(out)
			if len(violations) > 0 {
				return fmt.Errorf("test duration violations (maximum %s per package):\n%s",
					maxTestDuration, strings.Join(violations, "\n"))
			}
		}

		return fmt.Errorf("failed to run test duration check: %w", err)
	}

	violations := parseTestDurationOutput(out)
	if len(violations) > 0 {
		return fmt.Errorf("test duration violations (maximum %s per package):\n%s",
			maxTestDuration, strings.Join(violations, "\n"))
	}

	return nil
}

func parseTestDurationOutput(data []byte) []string {
	type testEvent struct {
		Action  string  `json:"Action"`
		Package string  `json:"Package"`
		Elapsed float64 `json:"Elapsed"`
	}

	var violations []string

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		var event testEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}

		if event.Action != "pass" && event.Action != "fail" {
			continue
		}

		if event.Package == "" || event.Elapsed == 0 {
			continue
		}

		elapsed := time.Duration(event.Elapsed * float64(time.Second))
		if elapsed > maxTestDuration {
			violations = append(violations,
				fmt.Sprintf("  - %s: %s (maximum %s)", event.Package, elapsed, maxTestDuration))
		}
	}

	return violations
}

func checkCoverage() error {
	log.Println("Checking code coverage (minimum 80% per package)...")

	tmpFile, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpFile.Close()

	defer os.Remove(tmpFile.Name())

	cmd := exec.Command("go", "test", fmt.Sprintf("-coverprofile=%s", tmpFile.Name()), "./...")

	var stdout bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run coverage check: %w", err)
	}

	goModData, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	modulePath := parseModulePath(string(goModData))

	violations, err := parseCoverageOutput(stdout.String(), modulePath)
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

func parseCoverageOutput(output, modulePath string) ([]string, error) {
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
			pkgName := matches[1]

			dir := packageToDir(pkgName, modulePath)
			if packageNeedsTests(dir) {
				violations = append(violations,
					fmt.Sprintf("  - %s: no test files", pkgName))
			}
		}
	}

	return violations, scanner.Err()
}

func packageToDir(pkgName, modulePath string) string {
	prefix := fmt.Sprintf("%s/", modulePath)
	if modulePath != "" && strings.HasPrefix(pkgName, prefix) {
		return strings.TrimPrefix(pkgName, prefix)
	}

	if pkgName == modulePath {
		return "."
	}

	return pkgName
}

func packageNeedsTests(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".pb.go") {
			continue
		}

		path := filepath.Join(dir, name)
		if !hasSkipDirective(path) && hasSignificantFunctions(path) {
			return true
		}
	}

	return false
}
