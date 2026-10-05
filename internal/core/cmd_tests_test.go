package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_runCommand(t *testing.T) {
	t.Run("runs successful command", func(t *testing.T) {
		err := runCommand("echo", "hello")

		assert.NoError(t, err)
	})

	t.Run("returns error for failed command", func(t *testing.T) {
		err := runCommand("false")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run")
	})

	t.Run("returns error for non-existent command", func(t *testing.T) {
		err := runCommand("nonexistent-command-xyz")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run")
	})

	t.Run("returns error on timeout", func(t *testing.T) {
		original := taskTimeout
		taskTimeout = 100 * time.Millisecond
		defer func() { taskTimeout = original }()

		err := runCommand("sleep", "10")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "task timed out")
	})
}

func Test_goTagsArgs(t *testing.T) {
	t.Run("empty tags returns no args", func(t *testing.T) {
		assert.Empty(t, goTagsArgs(nil))
		assert.Empty(t, goTagsArgs([]string{}))
	})

	t.Run("single tag", func(t *testing.T) {
		assert.Equal(t, []string{"-tags=integration"}, goTagsArgs([]string{"integration"}))
	})

	t.Run("each tag becomes its own flag", func(t *testing.T) {
		assert.Equal(t, []string{"-tags=integration", "-tags=e2e"}, goTagsArgs([]string{"integration", "e2e"}))
	})
}

func Test_goTestCommands(t *testing.T) {
	untagged := []command{
		{name: "go", args: []string{"fmt", "./..."}},
		{name: "go", args: []string{"vet", "./..."}},
		{name: "go", args: []string{"mod", "tidy", "-v"}},
		{name: "go", args: []string{"clean", "-testcache"}},
		{name: "go", args: []string{"test", "-cover", "./..."}},
		{name: "go", args: []string{"test", "-race", "./..."}},
	}

	t.Run("without tags runs only the untagged pass", func(t *testing.T) {
		assert.Equal(t, untagged, goTestCommands(nil))
	})

	t.Run("with tags keeps the untagged pass and appends a tagged pass", func(t *testing.T) {
		got := goTestCommands([]string{"integration", "e2e"})

		// The untagged run must always come first, unchanged.
		assert.Equal(t, untagged, got[:len(untagged)])

		// Followed by an additional tagged vet/test/race pass.
		assert.Equal(t, []command{
			{name: "go", args: []string{"vet", "-tags=integration", "-tags=e2e", "./..."}},
			{name: "go", args: []string{"test", "-cover", "-tags=integration", "-tags=e2e", "./..."}},
			{name: "go", args: []string{"test", "-race", "-tags=integration", "-tags=e2e", "./..."}},
		}, got[len(untagged):])
	})
}

func Test_runGoTests(t *testing.T) {
	t.Run("runs all commands in a valid go project", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.21\n"), 0644))
		require.NoError(t, os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n"), 0644))

		err := runGoTests(nil)

		assert.NoError(t, err)
	})

	t.Run("runs with build tags", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.21\n"), 0644))
		require.NoError(t, os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n"), 0644))

		err := runGoTests([]string{"integration", "e2e"})

		assert.NoError(t, err)
	})

	t.Run("returns error when command fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		// No go.mod means "go fmt ./..." will fail
		err := runGoTests(nil)

		assert.Error(t, err)
	})
}

func Test_runGoreleaserCheck(t *testing.T) {
	t.Run("skips when no goreleaser config", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		err := runGoreleaserCheck()

		assert.NoError(t, err)
	})

	t.Run("skips when goreleaser not installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile(".goreleaser.yml", []byte("builds: []\n"), 0644))

		origPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir)
		defer os.Setenv("PATH", origPath)

		err := runGoreleaserCheck()

		assert.NoError(t, err)
	})
}

func Test_runMarkdownlint(t *testing.T) {
	t.Run("skips when markdownlint not installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("README.md", []byte("# Title\n"), 0644))

		origPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir)
		defer os.Setenv("PATH", origPath)

		err := runMarkdownlint()

		assert.NoError(t, err)
	})

	t.Run("skips when no targets exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		binDir := writeFakeMarkdownlint(t, 0)

		origPath := os.Getenv("PATH")
		os.Setenv("PATH", binDir)
		defer os.Setenv("PATH", origPath)

		err := runMarkdownlint()

		assert.NoError(t, err)
	})

	t.Run("runs when installed and README.md exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("README.md", []byte("# Title\n"), 0644))

		binDir := writeFakeMarkdownlint(t, 0)

		origPath := os.Getenv("PATH")
		os.Setenv("PATH", binDir)
		defer os.Setenv("PATH", origPath)

		err := runMarkdownlint()

		assert.NoError(t, err)
	})

	t.Run("runs when installed and lowercase readme.md exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("readme.md", []byte("# Title\n"), 0644))

		binDir := writeFakeMarkdownlint(t, 0)

		origPath := os.Getenv("PATH")
		os.Setenv("PATH", binDir)
		defer os.Setenv("PATH", origPath)

		err := runMarkdownlint()

		assert.NoError(t, err)
	})

	t.Run("runs when installed and docs dir exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.Mkdir("docs", 0755))

		binDir := writeFakeMarkdownlint(t, 0)

		origPath := os.Getenv("PATH")
		os.Setenv("PATH", binDir)
		defer os.Setenv("PATH", origPath)

		err := runMarkdownlint()

		assert.NoError(t, err)
	})

	t.Run("passes a file once even when targets resolve to it twice", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		// On a case-insensitive filesystem os.Stat succeeds for both README.md
		// and readme.md even though a single file exists on disk; the file must
		// still be linted only once.
		require.NoError(t, os.WriteFile("README.md", []byte("# Title\n"), 0644))

		argsPath := filepath.Join(tmpDir, "markdownlint-args")
		binDir := writeRecordingMarkdownlint(t, argsPath)

		origPath := os.Getenv("PATH")
		os.Setenv("PATH", binDir)
		defer os.Setenv("PATH", origPath)

		require.NoError(t, runMarkdownlint())

		recorded, err := os.ReadFile(argsPath)
		require.NoError(t, err)

		lines := strings.Fields(string(recorded))
		seen := make(map[string]bool, len(lines))
		for _, line := range lines {
			assert.Falsef(t, seen[line], "%q passed to markdownlint more than once", line)
			seen[line] = true
		}

		assert.NotEmpty(t, lines, "expected at least one target to be linted")
	})

	t.Run("returns error when markdownlint fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("README.md", []byte("# Title\n"), 0644))

		binDir := writeFakeMarkdownlint(t, 1)

		origPath := os.Getenv("PATH")
		os.Setenv("PATH", binDir)
		defer os.Setenv("PATH", origPath)

		err := runMarkdownlint()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run")
	})
}

func Test_markdownlintSeen(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.md"), []byte("a\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.md"), []byte("b\n"), 0644))

	infoA, err := os.Stat(filepath.Join(tmpDir, "a.md"))
	require.NoError(t, err)
	infoASame, err := os.Stat(filepath.Join(tmpDir, "a.md"))
	require.NoError(t, err)
	infoB, err := os.Stat(filepath.Join(tmpDir, "b.md"))
	require.NoError(t, err)

	t.Run("empty seen set returns false", func(t *testing.T) {
		assert.False(t, markdownlintSeen(nil, infoA))
	})

	t.Run("detects the same file", func(t *testing.T) {
		assert.True(t, markdownlintSeen([]os.FileInfo{infoA}, infoASame))
	})

	t.Run("treats distinct files as unseen", func(t *testing.T) {
		assert.False(t, markdownlintSeen([]os.FileInfo{infoA}, infoB))
	})
}

// writeFakeMarkdownlint creates a stub markdownlint executable that exits with
// the given status code and returns the directory holding it.
func writeFakeMarkdownlint(t *testing.T, exitCode int) string {
	t.Helper()

	binDir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nexit %d\n", exitCode)
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "markdownlint"), []byte(script), 0755))

	return binDir
}

// writeRecordingMarkdownlint creates a stub markdownlint executable that writes
// each argument on its own line to argsPath and returns the directory holding
// it.
func writeRecordingMarkdownlint(t *testing.T, argsPath string) string {
	t.Helper()

	binDir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nfor a in \"$@\"; do echo \"$a\"; done > %q\n", argsPath)
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "markdownlint"), []byte(script), 0755))

	return binDir
}

func Test_runRustTests(t *testing.T) {
	t.Run("returns error when cargo is not available", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		origPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir)
		defer os.Setenv("PATH", origPath)

		err := runRustTests()

		assert.Error(t, err)
	})
}

func TestCreateTestsCommand(t *testing.T) {
	assertCommandBehavior(t, createTestsCommand, "tests", "Run tests with coverage, race detection, and linting")

	t.Run("returns error when go tests fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.21\n"), 0644))
		require.NoError(t, os.WriteFile("main.go", []byte("package main\n\nfunc main() { invalid }\n"), 0644))

		cmd := createTestsCommand()
		err := cmd.RunE(cmd, nil)
		assert.Error(t, err)
	})
}
