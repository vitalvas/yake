package core

import (
	"os"
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
