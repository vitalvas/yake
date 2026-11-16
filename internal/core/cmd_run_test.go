package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ensureProjectRoot(t *testing.T) {
	t.Run("passes when .git is present", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.Mkdir(".git", 0755))

		assert.NoError(t, ensureProjectRoot())
	})

	t.Run("passes when .git is a file", func(t *testing.T) {
		// Worktrees and submodules use a .git file rather than a directory.
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile(".git", []byte("gitdir: ../.git/worktrees/x\n"), 0644))

		assert.NoError(t, ensureProjectRoot())
	})

	t.Run("errors when .git is absent", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		err := ensureProjectRoot()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project root")
	})
}

func Test_runLocalValidators(t *testing.T) {
	setup := func(t *testing.T) {
		t.Helper()

		originalDir, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(originalDir) })

		os.Chdir(t.TempDir())
		require.NoError(t, os.MkdirAll(localValidatorsDir, 0755))
	}

	writeValidator := func(t *testing.T, name, script string, mode os.FileMode) {
		t.Helper()

		require.NoError(t, os.WriteFile(filepath.Join(localValidatorsDir, name), []byte(script), mode))
	}

	t.Run("no validators dir", func(t *testing.T) {
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(t.TempDir())

		assert.NoError(t, runLocalValidators())
	})

	t.Run("passing validator", func(t *testing.T) {
		setup(t)
		writeValidator(t, "ok.sh", "#!/bin/sh\necho fine\n", 0755)

		assert.NoError(t, runLocalValidators())
	})

	t.Run("failing validator reports stdout and stderr", func(t *testing.T) {
		setup(t)
		writeValidator(t, "bad.sh", "#!/bin/sh\necho out-text\necho err-text >&2\nexit 3\n", 0755)

		err := runLocalValidators()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad.sh")
		assert.Contains(t, err.Error(), "exit status 3")
		assert.Contains(t, err.Error(), "out-text")
		assert.Contains(t, err.Error(), "err-text")
	})

	t.Run("skips non-executable files and directories", func(t *testing.T) {
		setup(t)
		writeValidator(t, "README.md", "#!/bin/sh\nexit 1\n", 0644)
		require.NoError(t, os.Mkdir(filepath.Join(localValidatorsDir, "sub"), 0755))

		assert.NoError(t, runLocalValidators())
	})

	t.Run("stops at first failure in name order", func(t *testing.T) {
		setup(t)
		writeValidator(t, "a.sh", "#!/bin/sh\necho first\nexit 1\n", 0755)
		writeValidator(t, "b.sh", "#!/bin/sh\necho second\nexit 1\n", 0755)

		err := runLocalValidators()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "a.sh")
		assert.NotContains(t, err.Error(), "second")
	})
}

func BenchmarkRunLocalValidators(b *testing.B) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(b.TempDir())
	require.NoError(b, os.MkdirAll(localValidatorsDir, 0755))
	require.NoError(b, os.WriteFile(filepath.Join(localValidatorsDir, "ok.sh"), []byte("#!/bin/sh\n"), 0755))

	for b.Loop() {
		if err := runLocalValidators(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestCreateRunCommand(t *testing.T) {
	assertCommandBehavior(t, createRunCommand, "run", "Run tests and policy checks")

	t.Run("returns error when go tests fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.Mkdir(".git", 0755))
		require.NoError(t, os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.21\n"), 0644))
		require.NoError(t, os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n"), 0644))
		require.NoError(t, os.WriteFile("main_test.go", []byte("package main\n\nimport \"testing\"\n\nfunc TestFail(t *testing.T) { t.Fatal(\"boom\") }\n"), 0644))

		cmd := createRunCommand()
		err := cmd.RunE(cmd, nil)
		assert.Error(t, err)
	})
}
