package githook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommitMsg(t *testing.T) {
	writeMsg := func(t *testing.T, msg string) string {
		t.Helper()

		path := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
		require.NoError(t, os.WriteFile(path, []byte(msg), 0644))

		return path
	}

	tests := []struct {
		name    string
		msg     string
		wantErr string
	}{
		{
			name: "valid feat commit",
			msg:  "feat: add new feature",
		},
		{
			name: "valid fix with scope",
			msg:  "fix(parser): handle edge case",
		},
		{
			name: "valid deps commit",
			msg:  "deps: update dependencies",
		},
		{
			name: "valid perf commit",
			msg:  "perf: optimize query",
		},
		{
			name: "valid revert commit",
			msg:  "revert: undo last change",
		},
		{
			name: "valid docs commit",
			msg:  "docs: update readme",
		},
		{
			name: "valid chore commit",
			msg:  "chore: cleanup",
		},
		{
			name: "merge commit allowed",
			msg:  "Merge branch 'feature' into main",
		},
		{
			name: "non-conventional commit allowed",
			msg:  "just a regular commit message",
		},
		{
			name:    "denied type style",
			msg:     "style: format code",
			wantErr: "commit type 'style' is not allowed",
		},
		{
			name:    "denied type refactor",
			msg:     "refactor: restructure module",
			wantErr: "commit type 'refactor' is not allowed",
		},
		{
			name:    "denied type test",
			msg:     "test: add unit tests",
			wantErr: "commit type 'test' is not allowed",
		},
		{
			name:    "denied type build",
			msg:     "build: update makefile",
			wantErr: "commit type 'build' is not allowed",
		},
		{
			name:    "denied type ci",
			msg:     "ci: update workflow",
			wantErr: "commit type 'ci' is not allowed",
		},
		{
			name:    "breaking change not allowed",
			msg:     "feat!: breaking change",
			wantErr: "breaking change indicator (!) is not allowed",
		},
		{
			name:    "breaking change with scope not allowed",
			msg:     "feat(api)!: breaking change",
			wantErr: "breaking change indicator (!) is not allowed",
		},
		{
			name:    "multiline message not allowed",
			msg:     "feat: add feature\n\nsome body text",
			wantErr: "commit message must be a single line",
		},
		{
			name:    "multiline with only newline",
			msg:     "feat: first line\nsecond line",
			wantErr: "commit message must be a single line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeMsg(t, tt.msg)
			err := RunCommitMsg(path)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunCommitMsg_FileNotFound(t *testing.T) {
	err := RunCommitMsg("/nonexistent/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading commit message file")
}

func TestRunCommitMsg_LockCommit(t *testing.T) {
	setupLockTest := func(t *testing.T, lockContent []byte) (string, func()) {
		t.Helper()

		dir := t.TempDir()

		origDir, err := os.Getwd()
		require.NoError(t, err)

		require.NoError(t, os.Chdir(dir))
		require.NoError(t, os.MkdirAll(".git", 0755))
		require.NoError(t, os.WriteFile(".git/lock_commit", lockContent, 0644))

		msgPath := filepath.Join(dir, "COMMIT_EDITMSG")
		require.NoError(t, os.WriteFile(msgPath, []byte("feat: test"), 0644))

		return msgPath, func() {
			require.NoError(t, os.Chdir(origDir))
		}
	}

	t.Run("empty lock file", func(t *testing.T) {
		msgPath, cleanup := setupLockTest(t, nil)
		t.Cleanup(cleanup)

		err := RunCommitMsg(msgPath)
		require.Error(t, err)
		assert.Equal(t, "commit is locked", err.Error())
	})

	t.Run("lock file with message", func(t *testing.T) {
		msgPath, cleanup := setupLockTest(t, []byte("deploy in progress"))
		t.Cleanup(cleanup)

		err := RunCommitMsg(msgPath)
		require.Error(t, err)
		assert.Equal(t, "deploy in progress", err.Error())
	})
}
