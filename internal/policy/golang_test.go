package policy

import (
	"os"
	"path/filepath"
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
