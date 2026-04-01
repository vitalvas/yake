package core

import (
	"os"
	"testing"

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
}

func Test_runGoTests(t *testing.T) {
	t.Run("runs all commands in a valid go project", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.21\n"), 0644))
		require.NoError(t, os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n"), 0644))

		err := runGoTests()

		assert.NoError(t, err)
	})

	t.Run("returns error when command fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		// No go.mod means "go fmt ./..." will fail
		err := runGoTests()

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
