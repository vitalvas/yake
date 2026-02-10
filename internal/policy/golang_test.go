package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validTestFile = `package test

import "testing"

func TestExample(t *testing.T) {}
`

const testFileWithoutTestingImport = `package test

func TestExample() {}
`

const testFileWithoutFunctions = `package test

import "testing"

var _ = testing.T{}
`

func Test_validateTestFileName(t *testing.T) {
	t.Run("valid test file with source", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package test"), 0644))
		require.NoError(t, os.WriteFile("service_test.go", []byte(validTestFile), 0644))

		violations := validateTestFileName("service_test.go")
		assert.Empty(t, violations)
	})

	t.Run("valid e2e test file with source", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package test"), 0644))
		require.NoError(t, os.WriteFile("service_e2e_test.go", []byte(validTestFile), 0644))

		violations := validateTestFileName("service_e2e_test.go")
		assert.Empty(t, violations)
	})

	t.Run("test file missing source", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("orphan_test.go", []byte(validTestFile), 0644))

		violations := validateTestFileName("orphan_test.go")
		require.Len(t, violations, 1)
		assert.Contains(t, violations[0], "missing source file")
	})

	t.Run("test file missing testing import", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package test"), 0644))
		require.NoError(t, os.WriteFile("service_test.go", []byte(testFileWithoutTestingImport), 0644))

		violations := validateTestFileName("service_test.go")
		require.Len(t, violations, 1)
		assert.Contains(t, violations[0], "missing 'testing' package import")
	})

	t.Run("skips test file without functions", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("empty_test.go", []byte(testFileWithoutFunctions), 0644))

		violations := validateTestFileName("empty_test.go")
		assert.Empty(t, violations)
	})

	t.Run("invalid unit test naming pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package test"), 0644))
		require.NoError(t, os.WriteFile("service_unit_test.go", []byte(validTestFile), 0644))

		violations := validateTestFileName("service_unit_test.go")
		require.NotEmpty(t, violations)
		assert.Contains(t, violations[0], "invalid naming pattern")
	})

	t.Run("invalid bench test naming pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package test"), 0644))
		require.NoError(t, os.WriteFile("service_bench_test.go", []byte(validTestFile), 0644))

		violations := validateTestFileName("service_bench_test.go")
		require.NotEmpty(t, violations)
		assert.Contains(t, violations[0], "invalid naming pattern")
	})

	t.Run("invalid integration test naming pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package test"), 0644))
		require.NoError(t, os.WriteFile("service_integration_test.go", []byte(validTestFile), 0644))

		violations := validateTestFileName("service_integration_test.go")
		require.NotEmpty(t, violations)
		assert.Contains(t, violations[0], "invalid naming pattern")
	})

	t.Run("test file in subdirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.MkdirAll("pkg/service", 0755))
		require.NoError(t, os.WriteFile("pkg/service/handler.go", []byte("package service"), 0644))
		require.NoError(t, os.WriteFile("pkg/service/handler_test.go", []byte(validTestFile), 0644))

		violations := validateTestFileName(filepath.Join("pkg", "service", "handler_test.go"))
		assert.Empty(t, violations)
	})

	t.Run("collects multiple violations", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("orphan_test.go", []byte(testFileWithoutTestingImport), 0644))

		violations := validateTestFileName("orphan_test.go")
		require.Len(t, violations, 2)
	})
}

func Test_validateSourceFile(t *testing.T) {
	t.Run("valid source file with test", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package main\nfunc Foo() {}"), 0644))
		require.NoError(t, os.WriteFile("service_test.go", []byte(validTestFile), 0644))

		violations := validateSourceFile("service.go")
		assert.Empty(t, violations)
	})

	t.Run("source file missing test", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("auth_cache.go", []byte("package main\nfunc Foo() {}"), 0644))

		violations := validateSourceFile("auth_cache.go")
		require.Len(t, violations, 1)
		assert.Contains(t, violations[0], "missing test file")
		assert.Contains(t, violations[0], "auth_cache_test.go")
	})

	t.Run("source file with skip directive bypasses test requirement", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		code := `//yake:skip-test
package main

func Foo() {}
`
		require.NoError(t, os.WriteFile("skipped.go", []byte(code), 0644))

		violations := validateSourceFile("skipped.go")
		assert.Empty(t, violations)
	})
}

func Test_hasTestingImport(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"returns true when testing is imported", validTestFile, true},
		{"returns false when testing is not imported", testFileWithoutTestingImport, false},
		{"returns false for invalid go file", "not valid go code {{{", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "example.go")
			require.NoError(t, os.WriteFile(filePath, []byte(tt.content), 0644))
			assert.Equal(t, tt.expected, hasTestingImport(filePath))
		})
	}

	t.Run("returns false for non-existent file", func(t *testing.T) {
		assert.False(t, hasTestingImport("/non/existent/file.go"))
	})
}

func Test_hasFunctions(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"returns true when file has functions", validTestFile, true},
		{"returns false when file has no functions", testFileWithoutFunctions, false},
		{"returns false for invalid go file", "not valid go code {{{", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "example.go")
			require.NoError(t, os.WriteFile(filePath, []byte(tt.content), 0644))
			assert.Equal(t, tt.expected, hasFunctions(filePath))
		})
	}

	t.Run("returns false for non-existent file", func(t *testing.T) {
		assert.False(t, hasFunctions("/non/existent/file.go"))
	})
}

func Test_hasSkipDirective(t *testing.T) {
	t.Run("returns true when skip directive present", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "main.go")

		code := `//yake:skip-test
package main

func main() {
	Execute()
}
`
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		assert.True(t, hasSkipDirective(filePath))
	})

	t.Run("returns true when skip directive after other comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "main.go")

		code := `// Some comment
//yake:skip-test
package main

func main() {}
`
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		assert.True(t, hasSkipDirective(filePath))
	})

	t.Run("returns false when skip directive after package", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "main.go")

		code := `package main

//yake:skip-test
func main() {}
`
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		assert.False(t, hasSkipDirective(filePath))
	})

	t.Run("returns false when no skip directive", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "main.go")

		code := `package main

func main() {}
`
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		assert.False(t, hasSkipDirective(filePath))
	})

	t.Run("returns false for non-existent file", func(t *testing.T) {
		assert.False(t, hasSkipDirective("/non/existent/file.go"))
	})
}

func Test_hasSignificantFunctions(t *testing.T) {
	t.Run("returns true for function with more than 5 lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "service.go")

		code := `package main

func Process() error {
	x := 1
	y := 2
	z := x + y
	w := z * 2
	v := w + 1
	return nil
}
`
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		assert.True(t, hasSignificantFunctions(filePath))
	})

	t.Run("returns false for function with 5 or less lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "main.go")

		code := `package main

func main() {
	core.Execute()
}
`
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		assert.False(t, hasSignificantFunctions(filePath))
	})

	t.Run("returns false for file with only structs", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "models.go")

		code := `package main

type User struct {
	ID int
}
`
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		assert.False(t, hasSignificantFunctions(filePath))
	})

	t.Run("returns false for non-existent file", func(t *testing.T) {
		assert.False(t, hasSignificantFunctions("/non/existent/file.go"))
	})
}

func generateLargeFunc(name string, lines int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("func %s() {\n", name))

	for i := range lines {
		b.WriteString(fmt.Sprintf("\t_ = %d\n", i))
	}

	b.WriteString("}\n")

	return b.String()
}

func generateLargeMethod(typeName, methodName string, lines int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("func (t *%s) %s() {\n", typeName, methodName))

	for i := range lines {
		b.WriteString(fmt.Sprintf("\t_ = %d\n", i))
	}

	b.WriteString("}\n")

	return b.String()
}

func Test_largeFunctions(t *testing.T) {
	t.Run("returns funcInfo for function with more than 25 lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "service.go")

		code := "package main\n\n" + generateLargeFunc("ProcessData", 30)
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		result := largeFunctions(filePath)
		require.Len(t, result, 1)
		assert.Equal(t, "ProcessData", result[0].Name)
		assert.Equal(t, 30, result[0].Lines)
		assert.Greater(t, result[0].StartLine, 0)
		assert.Greater(t, result[0].EndLine, result[0].StartLine)
	})

	t.Run("returns empty for function with 25 or fewer lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "service.go")

		code := "package main\n\n" + generateLargeFunc("SmallFunc", 25)
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		result := largeFunctions(filePath)
		assert.Empty(t, result)
	})

	t.Run("returns Type_Method format for methods", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "service.go")

		code := "package main\n\ntype Service struct{}\n\n" + generateLargeMethod("Service", "Handle", 30)
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		result := largeFunctions(filePath)
		require.Len(t, result, 1)
		assert.Equal(t, "Service_Handle", result[0].Name)
		assert.Equal(t, 30, result[0].Lines)
	})

	t.Run("returns multiple large functions", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "service.go")

		code := "package main\n\n" + generateLargeFunc("FuncA", 30) + "\n" + generateLargeFunc("FuncB", 30)
		require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

		result := largeFunctions(filePath)
		require.Len(t, result, 2)
		assert.Equal(t, "FuncA", result[0].Name)
		assert.Equal(t, "FuncB", result[1].Name)
	})

	t.Run("returns nil for non-existent file", func(t *testing.T) {
		result := largeFunctions("/non/existent/file.go")
		assert.Nil(t, result)
	})
}

func Test_parseCoverProfile(t *testing.T) {
	t.Run("parses valid profile with module prefix stripped", func(t *testing.T) {
		profile := `mode: set
github.com/example/app/internal/service/handler.go:10.30,20.2 5 1
github.com/example/app/internal/service/handler.go:22.40,35.2 7 0
github.com/example/app/main.go:5.15,10.2 3 1
`
		result := parseCoverProfile(profile, "github.com/example/app")

		require.Len(t, result, 2)
		require.Len(t, result["internal/service/handler.go"], 2)
		assert.Equal(t, 10, result["internal/service/handler.go"][0].StartLine)
		assert.Equal(t, 20, result["internal/service/handler.go"][0].EndLine)
		assert.Equal(t, 1, result["internal/service/handler.go"][0].Count)
		assert.Equal(t, 0, result["internal/service/handler.go"][1].Count)
		require.Len(t, result["main.go"], 1)
		assert.Equal(t, 1, result["main.go"][0].Count)
	})

	t.Run("returns empty map for empty profile", func(t *testing.T) {
		result := parseCoverProfile("", "github.com/example/app")
		assert.Empty(t, result)
	})

	t.Run("returns empty map for mode-only profile", func(t *testing.T) {
		result := parseCoverProfile("mode: set\n", "github.com/example/app")
		assert.Empty(t, result)
	})

	t.Run("handles empty module path", func(t *testing.T) {
		profile := `mode: set
mypackage/handler.go:10.30,20.2 5 1
`
		result := parseCoverProfile(profile, "")

		require.Len(t, result, 1)
		require.Len(t, result["mypackage/handler.go"], 1)
	})
}

func Test_parseModulePath(t *testing.T) {
	t.Run("extracts module path from go.mod", func(t *testing.T) {
		goMod := `module github.com/example/app

go 1.21

require (
	github.com/stretchr/testify v1.9.0
)
`
		assert.Equal(t, "github.com/example/app", parseModulePath(goMod))
	})

	t.Run("returns empty string for empty content", func(t *testing.T) {
		assert.Equal(t, "", parseModulePath(""))
	})

	t.Run("returns empty string when no module line", func(t *testing.T) {
		assert.Equal(t, "", parseModulePath("go 1.21\n"))
	})
}

func Test_isFuncCovered(t *testing.T) {
	t.Run("returns true when block with count > 0 is in range", func(t *testing.T) {
		blocks := []coverBlock{
			{StartLine: 12, EndLine: 20, Count: 1},
		}
		fn := funcInfo{Name: "Process", Lines: 30, StartLine: 10, EndLine: 42}
		assert.True(t, isFuncCovered(blocks, fn))
	})

	t.Run("returns false when all blocks have count 0", func(t *testing.T) {
		blocks := []coverBlock{
			{StartLine: 12, EndLine: 20, Count: 0},
			{StartLine: 22, EndLine: 30, Count: 0},
		}
		fn := funcInfo{Name: "Process", Lines: 30, StartLine: 10, EndLine: 42}
		assert.False(t, isFuncCovered(blocks, fn))
	})

	t.Run("returns false when no blocks exist", func(t *testing.T) {
		fn := funcInfo{Name: "Process", Lines: 30, StartLine: 10, EndLine: 42}
		assert.False(t, isFuncCovered(nil, fn))
	})

	t.Run("returns false when block is outside function range", func(t *testing.T) {
		blocks := []coverBlock{
			{StartLine: 50, EndLine: 60, Count: 1},
		}
		fn := funcInfo{Name: "Process", Lines: 30, StartLine: 10, EndLine: 42}
		assert.False(t, isFuncCovered(blocks, fn))
	})
}

func Test_findUncoveredLargeFunctions(t *testing.T) {
	t.Run("detects uncovered large function", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		goMod := "module testproject\n\ngo 1.21\n"
		require.NoError(t, os.WriteFile("go.mod", []byte(goMod), 0644))

		code := "package main\n\n" + generateLargeFunc("BigProcess", 30)
		require.NoError(t, os.WriteFile("service.go", []byte(code), 0644))

		// Coverage profile with no coverage for service.go
		profile := "mode: set\ntestproject/other.go:1.10,5.2 3 1\n"
		profilePath := filepath.Join(tmpDir, "cover.out")
		require.NoError(t, os.WriteFile(profilePath, []byte(profile), 0644))

		violations, err := findUncoveredLargeFunctions(profilePath)
		require.NoError(t, err)
		require.Len(t, violations, 1)
		assert.Contains(t, violations[0], "BigProcess")
		assert.Contains(t, violations[0], "no test coverage")
	})

	t.Run("passes when large function is covered", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		goMod := "module testproject\n\ngo 1.21\n"
		require.NoError(t, os.WriteFile("go.mod", []byte(goMod), 0644))

		code := "package main\n\n" + generateLargeFunc("BigProcess", 30)
		require.NoError(t, os.WriteFile("service.go", []byte(code), 0644))

		// Coverage profile with coverage inside the function body
		// Function starts at line 3 (func header), body starts at line 3, ends at line 34
		profile := "mode: set\ntestproject/service.go:4.10,33.2 30 1\n"
		profilePath := filepath.Join(tmpDir, "cover.out")
		require.NoError(t, os.WriteFile(profilePath, []byte(profile), 0644))

		violations, err := findUncoveredLargeFunctions(profilePath)
		require.NoError(t, err)
		assert.Empty(t, violations)
	})

	t.Run("skips files with skip directive", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		goMod := "module testproject\n\ngo 1.21\n"
		require.NoError(t, os.WriteFile("go.mod", []byte(goMod), 0644))

		code := "//yake:skip-test\npackage main\n\n" + generateLargeFunc("BigProcess", 30)
		require.NoError(t, os.WriteFile("service.go", []byte(code), 0644))

		profile := "mode: set\n"
		profilePath := filepath.Join(tmpDir, "cover.out")
		require.NoError(t, os.WriteFile(profilePath, []byte(profile), 0644))

		violations, err := findUncoveredLargeFunctions(profilePath)
		require.NoError(t, err)
		assert.Empty(t, violations)
	})

	t.Run("skips test files", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		goMod := "module testproject\n\ngo 1.21\n"
		require.NoError(t, os.WriteFile("go.mod", []byte(goMod), 0644))

		code := "package main\n\nimport \"testing\"\n\n" + generateLargeFunc("TestBig", 30)
		require.NoError(t, os.WriteFile("service_test.go", []byte(code), 0644))

		profile := "mode: set\n"
		profilePath := filepath.Join(tmpDir, "cover.out")
		require.NoError(t, os.WriteFile(profilePath, []byte(profile), 0644))

		violations, err := findUncoveredLargeFunctions(profilePath)
		require.NoError(t, err)
		assert.Empty(t, violations)
	})
}

func Test_parseCoverageOutput(t *testing.T) {
	t.Run("parses coverage above threshold", func(t *testing.T) {
		output := "ok  \tgithub.com/example/pkg\t0.005s\tcoverage: 85.0% of statements"

		violations, err := parseCoverageOutput(output)

		require.NoError(t, err)
		assert.Empty(t, violations)
	})

	t.Run("detects coverage below threshold", func(t *testing.T) {
		output := "ok  \tgithub.com/example/pkg\t0.005s\tcoverage: 50.0% of statements"

		violations, err := parseCoverageOutput(output)

		require.NoError(t, err)
		require.Len(t, violations, 1)
		assert.Contains(t, violations[0], "50.0%")
		assert.Contains(t, violations[0], "github.com/example/pkg")
	})

	t.Run("detects no test files", func(t *testing.T) {
		output := "?\tgithub.com/example/nopkg\t[no test files]"

		violations, err := parseCoverageOutput(output)

		require.NoError(t, err)
		require.Len(t, violations, 1)
		assert.Contains(t, violations[0], "no test files")
		assert.Contains(t, violations[0], "github.com/example/nopkg")
	})

	t.Run("handles multiple packages", func(t *testing.T) {
		output := `ok  	github.com/example/pkg1	0.005s	coverage: 90.0% of statements
ok  	github.com/example/pkg2	0.003s	coverage: 75.0% of statements
?	github.com/example/pkg3	[no test files]
ok  	github.com/example/pkg4	0.002s	coverage: 100.0% of statements`

		violations, err := parseCoverageOutput(output)

		require.NoError(t, err)
		assert.Len(t, violations, 2)
	})

	t.Run("handles empty output", func(t *testing.T) {
		violations, err := parseCoverageOutput("")

		require.NoError(t, err)
		assert.Empty(t, violations)
	})

	t.Run("handles cached coverage output", func(t *testing.T) {
		output := "ok  \tgithub.com/example/pkg\t(cached)\tcoverage: 75.0% of statements"

		violations, err := parseCoverageOutput(output)

		require.NoError(t, err)
		require.Len(t, violations, 1)
		assert.Contains(t, violations[0], "75.0%")
	})
}

func Test_checkTestFileNaming(t *testing.T) {
	t.Run("passes with valid test files", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package main"), 0644))
		require.NoError(t, os.WriteFile("service_test.go", []byte(validTestFile), 0644))
		require.NoError(t, os.WriteFile("handler.go", []byte("package main"), 0644))
		require.NoError(t, os.WriteFile("handler_e2e_test.go", []byte(validTestFile), 0644))

		err := checkTestFileNaming()
		assert.NoError(t, err)
	})

	t.Run("fails with orphan test file", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("orphan_test.go", []byte(validTestFile), 0644))

		err := checkTestFileNaming()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing source file")
	})

	t.Run("fails with test file missing testing import", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package main"), 0644))
		require.NoError(t, os.WriteFile("service_test.go", []byte(testFileWithoutTestingImport), 0644))

		err := checkTestFileNaming()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'testing' package import")
	})

	t.Run("skips vendor directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.MkdirAll("vendor/pkg", 0755))
		require.NoError(t, os.WriteFile("vendor/pkg/orphan_test.go", []byte("package pkg"), 0644))

		err := checkTestFileNaming()
		assert.NoError(t, err)
	})

	t.Run("skips .git directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.MkdirAll(".git/hooks", 0755))
		require.NoError(t, os.WriteFile(".git/hooks/orphan_test.go", []byte("package hooks"), 0644))

		err := checkTestFileNaming()
		assert.NoError(t, err)
	})

	t.Run("skips test directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.MkdirAll("test", 0755))
		require.NoError(t, os.WriteFile("test/orphan_test.go", []byte(validTestFile), 0644))

		err := checkTestFileNaming()
		assert.NoError(t, err)
	})

	t.Run("skips tests directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.MkdirAll("tests", 0755))
		require.NoError(t, os.WriteFile("tests/orphan_test.go", []byte(validTestFile), 0644))

		err := checkTestFileNaming()
		assert.NoError(t, err)
	})

	t.Run("skips non-go files", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("readme.md", []byte("# Test"), 0644))
		require.NoError(t, os.WriteFile("config.yaml", []byte("key: value"), 0644))

		err := checkTestFileNaming()
		assert.NoError(t, err)
	})

	t.Run("fails with invalid naming pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("service.go", []byte("package main"), 0644))
		require.NoError(t, os.WriteFile("service_unit_test.go", []byte(validTestFile), 0644))

		err := checkTestFileNaming()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid naming pattern")
	})

	t.Run("fails when non-test file imports testing", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("tests.go", []byte(validTestFile), 0644))

		err := checkTestFileNaming()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "imports 'testing' but is not named '{origin}_test.go' or '{origin}_e2e_test.go'")
	})

	t.Run("fails when source file with functions missing test file", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		sourceCode := `package main

func AuthCache() error {
	cache := make(map[string]string)
	cache["key"] = "value"
	cache["key2"] = "value2"
	cache["key3"] = "value3"
	_ = cache
	return nil
}
`
		require.NoError(t, os.WriteFile("auth_cache.go", []byte(sourceCode), 0644))

		err := checkTestFileNaming()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing test file")
		assert.Contains(t, err.Error(), "auth_cache_test.go")
	})

	t.Run("skips source file with only structs no functions", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		sourceCode := `package main

type User struct {
	ID   int
	Name string
}

type Config struct {
	Host string
	Port int
}
`
		require.NoError(t, os.WriteFile("models.go", []byte(sourceCode), 0644))

		err := checkTestFileNaming()
		assert.NoError(t, err)
	})

	t.Run("skips source file with small functions less than 5 lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		sourceCode := `package main

func main() {
	core.Execute()
}
`
		require.NoError(t, os.WriteFile("main.go", []byte(sourceCode), 0644))

		err := checkTestFileNaming()
		assert.NoError(t, err)
	})

	t.Run("skips source file with skip directive", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		sourceCode := `//yake:skip-test
package main

func Process() error {
	x := 1
	y := 2
	z := x + y
	w := z * 2
	v := w + 1
	_ = v
	return nil
}
`
		require.NoError(t, os.WriteFile("processor.go", []byte(sourceCode), 0644))

		err := checkTestFileNaming()
		assert.NoError(t, err)
	})

}

func TestRunGolangChecks(t *testing.T) {
	t.Run("passes with valid go project", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		createTestGoProject(t, tmpDir, 100)

		err := RunGolangChecks()
		assert.NoError(t, err)
	})

	t.Run("fails with low coverage", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		createTestGoProject(t, tmpDir, 50)

		err := RunGolangChecks()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "coverage violations")
	})
}

func Test_checkCoverage(t *testing.T) {
	t.Run("passes with high coverage", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		createTestGoProject(t, tmpDir, 100)

		err := checkCoverage()
		assert.NoError(t, err)
	})

	t.Run("fails with low coverage", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		createTestGoProject(t, tmpDir, 50)

		err := checkCoverage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "coverage violations")
	})

	t.Run("detects large uncovered function", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		createTestGoProjectWithLargeFunc(t, tmpDir)

		err := checkCoverage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no test coverage")
		assert.Contains(t, err.Error(), "BigUntested")
	})
}

func createTestGoProject(t *testing.T, dir string, coveragePercent int) {
	t.Helper()

	goMod := `module testproject

go 1.21
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644))

	mainGo := `package main

func Add(a, b int) int {
	return a + b
}

func Subtract(a, b int) int {
	return a - b
}

func main() {}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644))

	var testGo string

	if coveragePercent >= 80 {
		testGo = `package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("expected 5, got %d", result)
	}
}

func TestSubtract(t *testing.T) {
	result := Subtract(5, 3)
	if result != 2 {
		t.Errorf("expected 2, got %d", result)
	}
}
`
	} else {
		testGo = `package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("expected 5, got %d", result)
	}
}
`
	}

	require.NoError(t, os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(testGo), 0644))
}

func createTestGoProjectWithLargeFunc(t *testing.T, dir string) {
	t.Helper()

	goMod := `module testproject

go 1.21
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644))

	var b strings.Builder

	b.WriteString("package main\n\n")
	b.WriteString("func Add(a, b int) int { return a + b }\n\n")
	b.WriteString("func BigUntested() int {\n")

	for i := range 30 {
		b.WriteString(fmt.Sprintf("\t_ = %d\n", i))
	}

	b.WriteString("\treturn 0\n}\n\nfunc main() {}\n")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(b.String()), 0644))

	testGo := `package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("expected 5, got %d", result)
	}
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(testGo), 0644))
}
