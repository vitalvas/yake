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
	MinCoveragePercent = 80.0
	minFunctionLines   = 5
	skipDirective      = "//yake:skip-test"
)

func RunGolangChecks() error {
	log.Println("Running Go policy checks...")

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	var allErrors []string

	if cfg.Policy.EntryPoints.isEnabled() {
		if err := checkEntryPoints(cfg.Policy.EntryPoints.getMaxMainLines()); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.PackageNaming.isEnabled() {
		if err := checkPackageNaming(cfg.Policy.PackageNaming.getPattern()); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.StringConcat.isEnabled() {
		if err := checkStringConcat(); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.StdlibWrappers.isEnabled() {
		if err := checkStdlibWrappers(); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.FuncSignature.isEnabled() {
		if err := checkFuncSignature(cfg.Policy.FuncSignature.getMaxParams(), cfg.Policy.FuncSignature.getMaxResults()); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.CompositeLiteral.isEnabled() {
		if err := checkCompositeLiteral(cfg.Policy.CompositeLiteral.getMaxSingleLineFields()); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.Stuttering.isEnabled() {
		if err := checkStuttering(); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.GetterNaming.isEnabled() {
		if err := checkGetterNaming(); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.PrivateExportedMethods.isEnabled() {
		if err := checkPrivateExportedMethods(); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.NoInit.isEnabled() {
		if err := checkNoInit(); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.TestFileNaming.isEnabled() {
		if err := checkTestFileNaming(); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.TestDuration.isEnabled() {
		if err := checkTestDuration(cfg.Policy.TestDuration.getMaxDuration()); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if cfg.Policy.Coverage.isEnabled() {
		covOpts := coverageOptions{
			minCoverage:           cfg.Policy.Coverage.getMinCoverage(),
			maxUncoveredFuncLines: cfg.Policy.Coverage.getMaxUncoveredFuncLines(),
			excludePackages:       cfg.Policy.Coverage.getExcludePackages(),
			packageOverrides:      cfg.Policy.Coverage.getPackageOverrides(),
		}

		if err := checkCoverage(covOpts); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(allErrors, "\n"))
	}

	return nil
}

func checkEntryPoints(maxMainLines int) error {
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

	violations := validateMainFiles(mainFiles, maxMainLines)

	nonMainViolations := findNonMainEntryPoints()
	violations = append(violations, nonMainViolations...)

	if len(violations) > 0 {
		return fmt.Errorf("entry point violations:\n%s", strings.Join(violations, "\n"))
	}

	return nil
}

func findNonMainEntryPoints() []string {
	var violations []string

	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
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

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		if info.Name() == "main.go" || strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		if extractPackageName(path) == "main" {
			violations = append(violations,
				fmt.Sprintf("  - %s: package main only allowed in main.go files", path))
		}

		return nil
	})

	return violations
}

func validateMainFiles(paths []string, maxMainLines int) []string {
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

					if lines > maxMainLines {
						violations = append(violations,
							fmt.Sprintf("  - %s: main() is %d lines (maximum %d); move logic to internal/ or pkg/",
								path, lines, maxMainLines))
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

		if hasStringLit(binExpr) {
			pos := fset.Position(binExpr.OpPos)
			violations = append(violations,
				fmt.Sprintf("  - %s:%d: string concatenation with '+'", filePath, pos.Line))
			return false
		}

		return true
	})

	return violations
}

func hasStringLit(binExpr *ast.BinaryExpr) bool {
	return containsStringLit(binExpr.X) || containsStringLit(binExpr.Y)
}

func containsStringLit(expr ast.Expr) bool {
	if lit, ok := expr.(*ast.BasicLit); ok {
		return lit.Kind == token.STRING
	}

	if inner, ok := expr.(*ast.BinaryExpr); ok && inner.Op == token.ADD {
		return containsStringLit(inner.X) || containsStringLit(inner.Y)
	}

	return false
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

func checkFuncSignature(maxParams, maxResults int) error {
	log.Println("Checking function signature complexity...")

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

		fileViolations := findFuncSignatureViolations(path, maxParams, maxResults)
		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("function signature violations (use struct-based config):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func findFuncSignatureViolations(filePath string, maxParams, maxResults int) []string {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	if hasSkipDirective(filePath) {
		return nil
	}

	var violations []string

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || hasFuncSkipDirective(fn) {
			continue
		}

		paramCount := countFields(fn.Type.Params)
		if paramCount > maxParams {
			pos := fset.Position(fn.Pos())
			violations = append(violations,
				fmt.Sprintf("  - %s:%d: function '%s' has %d parameters (maximum %d)",
					filePath, pos.Line, fn.Name.Name, paramCount, maxParams))
		}

		resultCount := countFields(fn.Type.Results)
		if resultCount > maxResults {
			pos := fset.Position(fn.Pos())
			violations = append(violations,
				fmt.Sprintf("  - %s:%d: function '%s' has %d return values (maximum %d)",
					filePath, pos.Line, fn.Name.Name, resultCount, maxResults))
		}
	}

	return violations
}

func checkCompositeLiteral(maxSingleLineFields int) error {
	log.Println("Checking composite literal formatting...")

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

		fileViolations := findCompositeLiteralViolations(path, maxSingleLineFields)
		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("composite literal violations (each field must be on its own line):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func findCompositeLiteralViolations(filePath string, maxSingleLineFields int) []string {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	if hasSkipDirective(filePath) {
		return nil
	}

	var violations []string

	ast.Inspect(node, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		kvElts := 0
		for _, elt := range comp.Elts {
			if _, ok := elt.(*ast.KeyValueExpr); ok {
				kvElts++
			}
		}

		if kvElts <= 1 {
			return true
		}

		openLine := fset.Position(comp.Lbrace).Line
		closeLine := fset.Position(comp.Rbrace).Line

		if openLine == closeLine && kvElts <= maxSingleLineFields {
			return true
		}

		lineSet := make(map[int]int)
		for _, elt := range comp.Elts {
			line := fset.Position(elt.Pos()).Line
			lineSet[line]++
		}

		for line, count := range lineSet {
			if count > 1 {
				violations = append(violations,
					fmt.Sprintf("  - %s:%d: composite literal has %d fields on the same line",
						filePath, line, count))
			}
		}

		return true
	})

	return violations
}

func countFields(fields *ast.FieldList) int {
	if fields == nil {
		return 0
	}

	count := 0

	for _, field := range fields.List {
		if len(field.Names) == 0 {
			count++
		} else {
			count += len(field.Names)
		}
	}

	return count
}

func checkPackageNaming(pattern string) error {
	log.Println("Checking package naming conventions...")

	pkgNameRegex := regexp.MustCompile(pattern)

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

		if !pkgNameRegex.MatchString(pkgName) {
			violations = append(violations,
				fmt.Sprintf("  - %s: package name '%s' does not match '%s'", path, pkgName, pattern))
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

func checkStuttering() error {
	log.Println("Checking for stuttering in exported identifiers...")

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

		fileViolations := findStutteringViolations(path)
		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("stuttering violations (exported names should not repeat the package name):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func findStutteringViolations(filePath string) []string {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	pkgName := node.Name.Name
	if pkgName == "main" {
		return nil
	}

	pkgUpper := strings.ToUpper(pkgName[:1]) + pkgName[1:]

	if hasSkipDirective(filePath) {
		return nil
	}

	var violations []string

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			name := d.Name.Name
			if !d.Name.IsExported() || d.Recv != nil || hasFuncSkipDirective(d) {
				continue
			}

			if isStutteringName(name, pkgUpper) {
				pos := fset.Position(d.Pos())
				violations = append(violations,
					fmt.Sprintf("  - %s:%d: function '%s.%s' stutters", filePath, pos.Line, pkgName, name))
			}

		case *ast.GenDecl:
			if hasGenDeclSkipDirective(d) {
				continue
			}

			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					name := s.Name.Name
					if !s.Name.IsExported() {
						continue
					}

					if strings.EqualFold(name, pkgName) {
						continue
					}

					if isStutteringName(name, pkgUpper) {
						pos := fset.Position(s.Pos())
						violations = append(violations,
							fmt.Sprintf("  - %s:%d: type '%s.%s' stutters", filePath, pos.Line, pkgName, name))
					}

				case *ast.ValueSpec:
					for _, ident := range s.Names {
						if !ident.IsExported() {
							continue
						}

						if isStutteringName(ident.Name, pkgUpper) {
							pos := fset.Position(ident.Pos())
							violations = append(violations,
								fmt.Sprintf("  - %s:%d: identifier '%s.%s' stutters", filePath, pos.Line, pkgName, ident.Name))
						}
					}
				}
			}
		}
	}

	return violations
}

func isStutteringName(name, pkgUpper string) bool {
	if !strings.HasPrefix(name, pkgUpper) {
		return false
	}

	rest := name[len(pkgUpper):]
	if len(rest) == 0 {
		return false
	}

	return rest[0] >= 'A' && rest[0] <= 'Z'
}

func checkGetterNaming() error {
	log.Println("Checking for Get prefix in getter methods...")

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

		fileViolations := findGetterViolations(path)
		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("getter naming violations (use Name() instead of GetName()):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func findGetterViolations(filePath string) []string {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	if hasSkipDirective(filePath) {
		return nil
	}

	var violations []string

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || hasFuncSkipDirective(fn) {
			continue
		}

		name := fn.Name.Name
		if !fn.Name.IsExported() || !strings.HasPrefix(name, "Get") || len(name) <= 3 {
			continue
		}

		afterGet := name[3:]
		if len(afterGet) == 0 || afterGet[0] < 'A' || afterGet[0] > 'Z' {
			continue
		}

		paramCount := countFields(fn.Type.Params)
		resultCount := countFields(fn.Type.Results)

		if paramCount == 0 && resultCount == 1 {
			pos := fset.Position(fn.Pos())
			violations = append(violations,
				fmt.Sprintf("  - %s:%d: method '%s' should be '%s'", filePath, pos.Line, name, afterGet))
		}
	}

	return violations
}

// stdlibInterfaceMethods lists exported method names from standard library
// interfaces. Private structs are allowed to have these methods because they
// implement interfaces rather than exposing arbitrary public API.
// Generated from Go stdlib using go/parser to extract all exported interface methods.
var stdlibInterfaceMethods = map[string]bool{
	"Accept":                     true, // net.Listener
	"Add":                        true, // crypto/elliptic.Curve
	"Addr":                       true, // net.Listener
	"Align":                      true, // reflect.Type
	"Alignof":                    true, // go/types.Sizes
	"AppendBinary":               true, // encoding.BinaryAppender
	"AppendText":                 true, // encoding.TextAppender
	"AppendUint16":               true, // encoding/binary.AppendByteOrder
	"AppendUint32":               true, // encoding/binary.AppendByteOrder
	"AppendUint64":               true, // encoding/binary.AppendByteOrder
	"AssignableTo":               true, // reflect.Type
	"At":                         true, // image.Image
	"Begin":                      true, // database/sql/driver.Conn
	"BeginTx":                    true, // database/sql/driver.ConnBeginTx
	"Bits":                       true, // reflect.Type
	"BlockSize":                  true, // crypto/cipher.Block, hash.Hash
	"Bounds":                     true, // image.Image
	"Bytes":                      true, // crypto.Encapsulator
	"CanSeq":                     true, // reflect.Type
	"CanSeq2":                    true, // reflect.Type
	"ChanDir":                    true, // reflect.Type
	"CheckNamedValue":            true, // database/sql/driver.NamedValueChecker
	"Clone":                      true, // hash.Cloner
	"Close":                      true, // io.Closer, io/fs.File, net.Conn, database/sql/driver.Conn
	"CloseNotify":                true, // net/http.CloseNotifier
	"ColorIndexAt":               true, // image.PalettedImage
	"ColorModel":                 true, // image.Image
	"ColumnConverter":            true, // database/sql/driver.ColumnConverter
	"ColumnTypeDatabaseTypeName": true, // database/sql/driver.RowsColumnTypeDatabaseTypeName
	"ColumnTypeLength":           true, // database/sql/driver.RowsColumnTypeLength
	"ColumnTypeNullable":         true, // database/sql/driver.RowsColumnTypeNullable
	"ColumnTypePrecisionScale":   true, // database/sql/driver.RowsColumnTypePrecisionScale
	"ColumnTypeScanType":         true, // database/sql/driver.RowsColumnTypeScanType
	"Columns":                    true, // database/sql/driver.Rows
	"Commit":                     true, // database/sql/driver.Tx
	"Common":                     true, // debug/dwarf.Type
	"Comparable":                 true, // reflect.Type
	"Connect":                    true, // database/sql/driver.Connector
	"Control":                    true, // syscall.RawConn
	"Convert":                    true, // image/color.Model
	"ConvertValue":               true, // database/sql/driver.ValueConverter
	"ConvertibleTo":              true, // reflect.Type
	"Cookies":                    true, // net/http.CookieJar
	"Copy":                       true, // text/template/parse.Node
	"CryptBlocks":                true, // crypto/cipher.BlockMode
	"Deadline":                   true, // context.Context
	"Decapsulate":                true, // crypto.Decapsulator
	"Decrypt":                    true, // crypto.Decrypter, crypto/cipher.Block
	"Done":                       true, // context.Context
	"Double":                     true, // crypto/elliptic.Curve
	"Draw":                       true, // image/draw.Drawer
	"Driver":                     true, // database/sql/driver.Connector
	"ECDH":                       true, // crypto/ecdh.KeyExchanger
	"Elem":                       true, // reflect.Type
	"Enabled":                    true, // log/slog.Handler
	"Encapsulate":                true, // crypto.Encapsulator
	"Encapsulator":               true, // crypto.Decapsulator
	"Encrypt":                    true, // crypto/cipher.Block
	"End":                        true, // go/ast.Node
	"Err":                        true, // context.Context
	"Error":                      true, // error
	"Eval":                       true, // go/build/constraint.Expr
	"ExactString":                true, // go/constant.Value
	"Exec":                       true, // database/sql/driver.Execer
	"ExecContext":                true, // database/sql/driver.ExecerContext
	"Exported":                   true, // go/types.Object
	"Field":                      true, // reflect.Type
	"FieldAlign":                 true, // reflect.Type
	"FieldByIndex":               true, // reflect.Type
	"FieldByName":                true, // reflect.Type
	"FieldByNameFunc":            true, // reflect.Type
	"Fields":                     true, // reflect.Type
	"Flag":                       true, // fmt.State
	"Flush":                      true, // net/http.Flusher
	"Format":                     true, // fmt.Formatter
	"Generate":                   true, // testing/quick.Generator
	"GenerateKey":                true, // crypto/ecdh.Curve
	"Glob":                       true, // io/fs.GlobFS
	"GoString":                   true, // fmt.GoStringer
	"GobDecode":                  true, // encoding/gob.GobDecoder
	"GobEncode":                  true, // encoding/gob.GobEncoder
	"Handle":                     true, // log/slog.Handler
	"HasNextResultSet":           true, // database/sql/driver.RowsNextResultSet
	"HashFunc":                   true, // crypto.SignerOpts
	"Header":                     true, // net/http.ResponseWriter
	"Hijack":                     true, // net/http.Hijacker
	"ID":                         true, // crypto/hpke.AEAD
	"Id":                         true, // go/types.Object
	"Implements":                 true, // reflect.Type
	"Import":                     true, // go/types.Importer
	"ImportFrom":                 true, // go/types.ImporterFrom
	"In":                         true, // reflect.Type
	"Info":                       true, // io/fs.DirEntry
	"Ins":                        true, // reflect.Type
	"Int63":                      true, // math/rand.Source
	"IsDir":                      true, // io/fs.DirEntry, io/fs.FileInfo
	"IsOnCurve":                  true, // crypto/elliptic.Curve
	"IsValid":                    true, // database/sql/driver.Validator
	"IsVariadic":                 true, // reflect.Type
	"Key":                        true, // reflect.Type
	"Kind":                       true, // go/constant.Value, reflect.Type
	"LastInsertId":               true, // database/sql.Result
	"Len":                        true, // sort.Interface, reflect.Type
	"Less":                       true, // sort.Interface
	"Level":                      true, // log/slog.Leveler
	"LocalAddr":                  true, // net.Conn, net.PacketConn
	"Lock":                       true, // sync.Locker
	"LogValue":                   true, // log/slog.LogValuer
	"Lstat":                      true, // io/fs.ReadLinkFS
	"MarshalBinary":              true, // encoding.BinaryMarshaler
	"MarshalJSON":                true, // encoding/json.Marshaler
	"MarshalJSONTo":              true, // encoding/json/v2.MarshalerTo
	"MarshalText":                true, // encoding.TextMarshaler
	"MarshalXML":                 true, // encoding/xml.Marshaler
	"MarshalXMLAttr":             true, // encoding/xml.MarshalerAttr
	"Match":                      true, // regexp.MatchString pattern
	"MatchString":                true, // regexp pattern
	"Method":                     true, // reflect.Type
	"MethodByName":               true, // reflect.Type
	"Methods":                    true, // reflect.Type
	"ModTime":                    true, // io/fs.FileInfo
	"Mode":                       true, // io/fs.FileInfo
	"Name":                       true, // io/fs.DirEntry, io/fs.FileInfo, reflect.Type
	"Network":                    true, // net.Addr
	"NewPrivateKey":              true, // crypto/ecdh.Curve
	"NewPublicKey":               true, // crypto/ecdh.Curve
	"Next":                       true, // database/sql/driver.Rows, net/smtp.Auth
	"NextResultSet":              true, // database/sql/driver.RowsNextResultSet
	"NonceSize":                  true, // crypto/cipher.AEAD
	"NumField":                   true, // reflect.Type
	"NumIn":                      true, // reflect.Type
	"NumInput":                   true, // database/sql/driver.Stmt
	"NumMethod":                  true, // reflect.Type
	"NumOut":                     true, // reflect.Type
	"Offsetsof":                  true, // go/types.Sizes
	"Open":                       true, // io/fs.FS, net/http.FileSystem
	"OpenConnector":              true, // database/sql/driver.DriverContext
	"Out":                        true, // reflect.Type
	"Outs":                       true, // reflect.Type
	"OverflowComplex":            true, // reflect.Type
	"OverflowFloat":              true, // reflect.Type
	"OverflowInt":                true, // reflect.Type
	"OverflowUint":               true, // reflect.Type
	"Overhead":                   true, // crypto/cipher.AEAD
	"Params":                     true, // crypto/elliptic.Curve
	"Parent":                     true, // go/types.Object
	"Ping":                       true, // database/sql/driver.Pinger
	"Pkg":                        true, // go/types.Object
	"PkgPath":                    true, // reflect.Type
	"Pop":                        true, // container/heap.Interface
	"Pos":                        true, // go/ast.Node, go/types.Object
	"Position":                   true, // text/template/parse.Node
	"Precision":                  true, // fmt.State
	"Prepare":                    true, // database/sql/driver.Conn
	"PrepareContext":             true, // database/sql/driver.ConnPrepareContext
	"Public":                     true, // crypto.Decrypter, crypto.Signer
	"PublicKey":                  true, // crypto/ecdh.KeyExchanger
	"PublicSuffix":               true, // net/http/cookiejar.PublicSuffixList
	"Push":                       true, // container/heap.Interface, net/http.Pusher
	"Put":                        true, // crypto/tls.ClientSessionCache
	"PutUint16":                  true, // encoding/binary.ByteOrder
	"PutUint32":                  true, // encoding/binary.ByteOrder
	"PutUint64":                  true, // encoding/binary.ByteOrder
	"Quantize":                   true, // image/draw.Quantizer
	"Query":                      true, // database/sql/driver.Queryer
	"QueryContext":               true, // database/sql/driver.QueryerContext
	"RGBA":                       true, // image/color.Color
	"RGBA64At":                   true, // image.RGBA64Image
	"Read":                       true, // io.Reader, io/fs.File, net.Conn
	"ReadAt":                     true, // io.ReaderAt
	"ReadByte":                   true, // io.ByteReader
	"ReadDir":                    true, // io/fs.ReadDirFS, io/fs.ReadDirFile
	"ReadFile":                   true, // io/fs.ReadFileFS
	"ReadFrom":                   true, // io.ReaderFrom, net.PacketConn
	"ReadLink":                   true, // io/fs.ReadLinkFS
	"ReadRequestBody":            true, // net/rpc.ServerCodec
	"ReadRequestHeader":          true, // net/rpc.ServerCodec
	"ReadResponseBody":           true, // net/rpc.ClientCodec
	"ReadResponseHeader":         true, // net/rpc.ClientCodec
	"ReadRune":                   true, // io.RuneReader
	"Readdir":                    true, // net/http.File
	"RemoteAddr":                 true, // net.Conn
	"Reset":                      true, // hash.Hash, compress/flate.Resetter
	"ResetSession":               true, // database/sql/driver.SessionResetter
	"Rollback":                   true, // database/sql/driver.Tx
	"RoundTrip":                  true, // net/http.RoundTripper
	"RowsAffected":               true, // database/sql.Result
	"RuntimeError":               true, // runtime.Error
	"ScalarBaseMult":             true, // crypto/elliptic.Curve
	"ScalarMult":                 true, // crypto/elliptic.Curve
	"Scan":                       true, // database/sql.Scanner, fmt.Scanner
	"Seal":                       true, // crypto/cipher.AEAD
	"Seed":                       true, // math/rand.Source
	"Seek":                       true, // io.Seeker
	"ServeHTTP":                  true, // net/http.Handler
	"Set":                        true, // flag.Value, image/draw.Image
	"SetCookies":                 true, // net/http.CookieJar
	"SetDeadline":                true, // net.Conn, net.PacketConn
	"SetRGBA64":                  true, // image/draw.RGBA64Image
	"SetReadDeadline":            true, // net.Conn, net.PacketConn
	"SetWriteDeadline":           true, // net.Conn, net.PacketConn
	"Sign":                       true, // crypto.Signer
	"SignMessage":                true, // crypto.MessageSigner
	"Signal":                     true, // os.Signal
	"Size":                       true, // hash.Hash, io/fs.FileInfo, reflect.Type
	"Sizeof":                     true, // go/types.Sizes
	"SkipSpace":                  true, // fmt.ScanState
	"Start":                      true, // net/smtp.Auth
	"Stat":                       true, // io/fs.File, io/fs.StatFS, net/http.File
	"String":                     true, // fmt.Stringer
	"Sub":                        true, // io/fs.SubFS
	"Sum":                        true, // hash.Hash
	"Sum32":                      true, // hash.Hash32
	"Sum64":                      true, // hash.Hash64
	"Swap":                       true, // sort.Interface
	"Sys":                        true, // io/fs.FileInfo
	"SyscallConn":                true, // syscall.Conn
	"Temporary":                  true, // net.Error
	"Timeout":                    true, // net.Error
	"Token":                      true, // encoding/xml.TokenReader, fmt.ScanState
	"Type":                       true, // go/types.Object, io/fs.DirEntry
	"Uint16":                     true, // encoding/binary.ByteOrder
	"Uint32":                     true, // encoding/binary.ByteOrder
	"Uint64":                     true, // encoding/binary.ByteOrder, math/rand.Source64
	"Underlying":                 true, // go/types.Type
	"Unlock":                     true, // sync.Locker
	"UnmarshalBinary":            true, // encoding.BinaryUnmarshaler
	"UnmarshalJSON":              true, // encoding/json.Unmarshaler
	"UnmarshalJSONFrom":          true, // encoding/json/v2.UnmarshalerFrom
	"UnmarshalText":              true, // encoding.TextUnmarshaler
	"UnmarshalXML":               true, // encoding/xml.Unmarshaler
	"UnmarshalXMLAttr":           true, // encoding/xml.UnmarshalerAttr
	"UnreadByte":                 true, // io.ByteScanner
	"UnreadRune":                 true, // io.RuneScanner
	"Unwrap":                     true, // errors
	"Value":                      true, // context.Context, database/sql/driver.Valuer
	"Visit":                      true, // go/ast.Visitor
	"Width":                      true, // fmt.State
	"WithAttrs":                  true, // log/slog.Handler
	"WithGroup":                  true, // log/slog.Handler
	"Write":                      true, // io.Writer, net.Conn, net/http.ResponseWriter
	"WriteAt":                    true, // io.WriterAt
	"WriteByte":                  true, // io.ByteWriter
	"WriteHeader":                true, // net/http.ResponseWriter
	"WriteRequest":               true, // net/rpc.ClientCodec
	"WriteResponse":              true, // net/rpc.ServerCodec
	"WriteString":                true, // io.StringWriter
	"WriteTo":                    true, // io.WriterTo, net.PacketConn
	"XORKeyStream":               true, // crypto/cipher.Stream

	// proto (commonly used non-stdlib)
	"ProtoMessage":     true,
	"ProtoReflect":     true,
	"MarshalLogObject": true,
	"MarshalLogArray":  true,
}

func checkPrivateExportedMethods() error {
	log.Println("Checking for exported methods on private structs...")

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

		fileViolations := findPrivateExportedMethodViolations(path)
		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("private struct exported method violations (private structs should not have exported methods):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func findPrivateExportedMethodViolations(filePath string) []string {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	if hasSkipDirective(filePath) {
		return nil
	}

	privateTypes := collectPrivateTypes(node)

	var violations []string

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || hasFuncSkipDirective(fn) {
			continue
		}

		if !fn.Name.IsExported() {
			continue
		}

		if stdlibInterfaceMethods[fn.Name.Name] {
			continue
		}

		receiverName := extractReceiverTypeName(fn)
		if receiverName == "" || !privateTypes[receiverName] {
			continue
		}

		pos := fset.Position(fn.Pos())
		violations = append(violations,
			fmt.Sprintf("  - %s:%d: exported method '%s' on private struct '%s'",
				filePath, pos.Line, fn.Name.Name, receiverName))
	}

	return violations
}

func collectPrivateTypes(node *ast.File) map[string]bool {
	types := make(map[string]bool)

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			name := typeSpec.Name.Name
			if !typeSpec.Name.IsExported() {
				types[name] = true
			}
		}
	}

	return types
}

func extractReceiverTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}

	recvType := fn.Recv.List[0].Type

	if starExpr, ok := recvType.(*ast.StarExpr); ok {
		recvType = starExpr.X
	}

	// Handle generic receivers like *codec[T]
	if indexExpr, ok := recvType.(*ast.IndexExpr); ok {
		recvType = indexExpr.X
	}

	if indexListExpr, ok := recvType.(*ast.IndexListExpr); ok {
		recvType = indexListExpr.X
	}

	if ident, ok := recvType.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}

func checkNoInit() error {
	log.Println("Checking for init() functions...")

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

		fileViolations := findInitViolations(path)
		violations = append(violations, fileViolations...)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(violations) > 0 {
		return fmt.Errorf("init() function violations (init() is forbidden; use explicit constructors or wire setup from main()):\n%s",
			strings.Join(violations, "\n"))
	}

	return nil
}

func findInitViolations(filePath string) []string {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	if hasSkipDirective(filePath) {
		return nil
	}

	var violations []string

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || hasFuncSkipDirective(fn) {
			continue
		}

		if fn.Name.Name != "init" {
			continue
		}

		if countFields(fn.Type.Params) != 0 || countFields(fn.Type.Results) != 0 {
			continue
		}

		pos := fset.Position(fn.Pos())
		violations = append(violations,
			fmt.Sprintf("  - %s:%d: init() function is forbidden", filePath, pos.Line))
	}

	return violations
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
	if hasSkipDirective(testPath) {
		return nil
	}

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

// goOSNames contains all valid GOOS values recognized by Go's build system.
var goOSNames = map[string]bool{
	"aix":       true,
	"android":   true,
	"darwin":    true,
	"dragonfly": true,
	"freebsd":   true,
	"hurd":      true,
	"illumos":   true,
	"ios":       true,
	"js":        true,
	"linux":     true,
	"nacl":      true,
	"netbsd":    true,
	"openbsd":   true,
	"plan9":     true,
	"solaris":   true,
	"wasip1":    true,
	"windows":   true,
	"zos":       true,
}

// goArchNames contains all valid GOARCH values recognized by Go's build system.
var goArchNames = map[string]bool{
	"386":      true,
	"amd64":    true,
	"arm":      true,
	"arm64":    true,
	"loong64":  true,
	"mips":     true,
	"mips64":   true,
	"mips64le": true,
	"mipsle":   true,
	"ppc64":    true,
	"ppc64le":  true,
	"riscv64":  true,
	"s390x":    true,
	"wasm":     true,
}

// stripPlatformSuffix removes trailing _GOOS, _GOARCH, or _GOOS_GOARCH from a base name.
// Returns the stripped name and true if a suffix was removed.
func stripPlatformSuffix(base string) (string, bool) {
	parts := strings.Split(base, "_")

	if len(parts) >= 3 {
		osVal := parts[len(parts)-2]
		archVal := parts[len(parts)-1]

		if goOSNames[osVal] && goArchNames[archVal] {
			return strings.Join(parts[:len(parts)-2], "_"), true
		}
	}

	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		if goOSNames[last] || goArchNames[last] {
			return strings.Join(parts[:len(parts)-1], "_"), true
		}
	}

	return base, false
}

func validateSourceFile(sourcePath string) []string {
	if hasSkipDirective(sourcePath) {
		return nil
	}

	var violations []string

	filename := filepath.Base(sourcePath)
	dir := filepath.Dir(sourcePath)

	if filename == "main.go" && extractPackageName(sourcePath) == "main" {
		return violations
	}

	baseName := strings.TrimSuffix(filename, ".go")
	testFile := fmt.Sprintf("%s_test.go", baseName)
	testPath := filepath.Join(dir, testFile)

	if _, stripped := stripPlatformSuffix(baseName); stripped {
		return violations
	}

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

func hasGenDeclSkipDirective(decl *ast.GenDecl) bool {
	if decl.Doc == nil {
		return false
	}

	for _, comment := range decl.Doc.List {
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

func largeFunctions(filePath string, maxUncoveredFuncLines int) []funcInfo {
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

		if lines <= maxUncoveredFuncLines {
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

func findUncoveredLargeFunctions(profilePath string, maxUncoveredFuncLines int, excludePackages []string) ([]string, error) {
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

			if isDirExcluded(path, excludePackages) {
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

		funcs := largeFunctions(path, maxUncoveredFuncLines)
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

func checkTestDuration(maxDuration time.Duration) error {
	log.Printf("Checking test duration (maximum %s per package)...", maxDuration)

	cmd := exec.Command("go", "test", "-json", fmt.Sprintf("-timeout=%s", maxDuration), "./...")
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		if len(out) > 0 {
			violations := parseTestDurationOutput(out, maxDuration)
			if len(violations) > 0 {
				return fmt.Errorf("test duration violations (maximum %s per package):\n%s",
					maxDuration, strings.Join(violations, "\n"))
			}
		}

		return fmt.Errorf("failed to run test duration check: %w", err)
	}

	violations := parseTestDurationOutput(out, maxDuration)
	if len(violations) > 0 {
		return fmt.Errorf("test duration violations (maximum %s per package):\n%s",
			maxDuration, strings.Join(violations, "\n"))
	}

	return nil
}

func parseTestDurationOutput(data []byte, maxDuration time.Duration) []string {
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
		if elapsed > maxDuration {
			violations = append(violations,
				fmt.Sprintf("  - %s: %s (maximum %s)", event.Package, elapsed, maxDuration))
		}
	}

	return violations
}

type coverageOptions struct {
	minCoverage           float64
	maxUncoveredFuncLines int
	excludePackages       []string
	packageOverrides      map[string]float64
}

func checkCoverage(opts coverageOptions) error {
	log.Printf("Checking code coverage (minimum %.0f%% per package)...", opts.minCoverage)

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

	violations, err := parseCoverageOutput(stdout.String(), modulePath, opts)
	if err != nil {
		return err
	}

	funcViolations, err := findUncoveredLargeFunctions(tmpFile.Name(), opts.maxUncoveredFuncLines, opts.excludePackages)
	if err != nil {
		return err
	}

	violations = append(violations, funcViolations...)

	if len(violations) > 0 {
		return fmt.Errorf("coverage violations (minimum %.0f%%):\n%s",
			opts.minCoverage, strings.Join(violations, "\n"))
	}

	return nil
}

func parseCoverageOutput(output, modulePath string, opts coverageOptions) ([]string, error) {
	var violations []string

	coverageRegex := regexp.MustCompile(`ok\s+(\S+)\s+(?:[\d.]+s|\(cached\))\s+coverage:\s+([\d.]+)%`)
	noCoverageRegex := regexp.MustCompile(`\?\s+(\S+)\s+\[no test files\]`)

	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		if matches := coverageRegex.FindStringSubmatch(line); len(matches) == 3 {
			pkgName := matches[1]

			if isPackageExcluded(pkgName, modulePath, opts.excludePackages) {
				continue
			}

			coverage, err := strconv.ParseFloat(matches[2], 64)
			if err != nil {
				continue
			}

			minCov := packageMinCoverage(pkgName, modulePath, opts.minCoverage, opts.packageOverrides)

			if coverage < minCov {
				violations = append(violations,
					fmt.Sprintf("  - %s: %.1f%% coverage (minimum %.0f%%)", pkgName, coverage, minCov))
			}
		}

		if matches := noCoverageRegex.FindStringSubmatch(line); len(matches) == 2 {
			pkgName := matches[1]

			if isPackageExcluded(pkgName, modulePath, opts.excludePackages) {
				continue
			}

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

func isPackageExcluded(pkgName, modulePath string, excludePackages []string) bool {
	for _, pattern := range excludePackages {
		fullPkg := fmt.Sprintf("%s/%s", modulePath, pattern)

		if pkgName == fullPkg || strings.HasPrefix(pkgName, fmt.Sprintf("%s/", fullPkg)) {
			return true
		}

		if pkgName == pattern || strings.HasPrefix(pkgName, fmt.Sprintf("%s/", pattern)) {
			return true
		}
	}

	return false
}

func isDirExcluded(dir string, excludePackages []string) bool {
	for _, pattern := range excludePackages {
		if dir == pattern || strings.HasPrefix(dir, fmt.Sprintf("%s/", pattern)) {
			return true
		}
	}

	return false
}

func packageMinCoverage(pkgName, modulePath string, defaultMin float64, overrides map[string]float64) float64 {
	for pattern, minCov := range overrides {
		fullPkg := fmt.Sprintf("%s/%s", modulePath, pattern)

		if pkgName == fullPkg || pkgName == pattern {
			return minCov
		}
	}

	return defaultMin
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
