package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initTestGitRepo(t *testing.T, branch string) {
	t.Helper()

	cmds := [][]string{
		{"git", "init", "-b", branch},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		require.NoError(t, cmd.Run())
	}
}

func TestCreateLinterNewCommand(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := createLinterNewCommand()

		require.NotNil(t, cmd)
		assert.Equal(t, "linter-new", cmd.Use)
		assert.Equal(t, "Create a new linter configuration file", cmd.Short)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("has required lang flag", func(t *testing.T) {
		cmd := createLinterNewCommand()

		flag := cmd.Flags().Lookup("lang")
		require.NotNil(t, flag)
		assert.Equal(t, "l", flag.Shorthand)
	})

	t.Run("returns error for unsupported language", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cmd := createLinterNewCommand()
		cmd.SetArgs([]string{"--lang", "unsupported"})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported language")
	})

	t.Run("returns error when config already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		os.WriteFile(".golangci.yml", []byte(""), 0644)

		cmd := createLinterNewCommand()
		cmd.SetArgs([]string{"--lang", "go"})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestCreateGithubDependabotCommand(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := createGithubDependabotCommand()

		require.NotNil(t, cmd)
		assert.Equal(t, "github-dependabot", cmd.Use)
		assert.Equal(t, "Create GitHub Dependabot configuration", cmd.Short)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("has required lang flag", func(t *testing.T) {
		cmd := createGithubDependabotCommand()

		flag := cmd.Flags().Lookup("lang")
		require.NotNil(t, flag)
		assert.Equal(t, "l", flag.Shorthand)
	})

	t.Run("returns error for unsupported language", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cmd := createGithubDependabotCommand()
		cmd.SetArgs([]string{"--lang", "unsupported"})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported language")
	})

	t.Run("returns error when config already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		os.MkdirAll(".github", 0755)
		os.WriteFile(filepath.Join(".github", "dependabot.yml"), []byte(""), 0644)

		cmd := createGithubDependabotCommand()
		cmd.SetArgs([]string{"--lang", "go"})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestCodeSubcommands(t *testing.T) {
	t.Run("contains expected subcommands", func(t *testing.T) {
		require.GreaterOrEqual(t, len(codeSubcommands), 4)

		uses := make([]string, len(codeSubcommands))
		for i, cmd := range codeSubcommands {
			uses[i] = cmd.Use
		}

		assert.Contains(t, uses, "defaults")
		assert.Contains(t, uses, "linter-new")
		assert.Contains(t, uses, "github-dependabot")
		assert.Contains(t, uses, "github-release-please")
	})
}

func TestDefaultsCommand(t *testing.T) {
	t.Run("runs without error when no go.mod exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		var defaultsCmd *cobra.Command
		for _, cmd := range codeSubcommands {
			if cmd.Use == "defaults" {
				defaultsCmd = cmd
				break
			}
		}

		require.NotNil(t, defaultsCmd)
		err := defaultsCmd.RunE(defaultsCmd, nil)
		assert.NoError(t, err)
	})

	t.Run("creates linter config when go.mod exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		os.WriteFile("go.mod", []byte("module test"), 0644)

		var defaultsCmd *cobra.Command
		for _, cmd := range codeSubcommands {
			if cmd.Use == "defaults" {
				defaultsCmd = cmd
				break
			}
		}

		require.NotNil(t, defaultsCmd)
		err := defaultsCmd.RunE(defaultsCmd, nil)
		assert.NoError(t, err)

		_, statErr := os.Stat(".golangci.yml")
		assert.NoError(t, statErr)
	})

	t.Run("skips linter config when already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		os.WriteFile("go.mod", []byte("module test"), 0644)
		os.WriteFile(".golangci.yml", []byte("existing"), 0644)

		var defaultsCmd *cobra.Command
		for _, cmd := range codeSubcommands {
			if cmd.Use == "defaults" {
				defaultsCmd = cmd
				break
			}
		}

		require.NotNil(t, defaultsCmd)
		err := defaultsCmd.RunE(defaultsCmd, nil)
		assert.NoError(t, err)

		content, _ := os.ReadFile(".golangci.yml")
		assert.Equal(t, "existing", string(content))
	})
}

func TestCodeLinterNewGolang(t *testing.T) {
	t.Run("creates golangci config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		err := codeLinterNewGolang()
		assert.NoError(t, err)

		_, statErr := os.Stat(".golangci.yml")
		assert.NoError(t, statErr)
	})
}

func TestCodeGithubDependabot(t *testing.T) {
	t.Run("creates dependabot config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		err := codeGithubDependabot("go")
		assert.NoError(t, err)

		_, statErr := os.Stat(".github/dependabot.yml")
		assert.NoError(t, statErr)
	})

	t.Run("creates .github directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		err := codeGithubDependabot("go")
		assert.NoError(t, err)

		info, statErr := os.Stat(".github")
		assert.NoError(t, statErr)
		assert.True(t, info.IsDir())
	})
}

func TestLinterNewCommandSuccess(t *testing.T) {
	t.Run("creates config for go language", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cmd := createLinterNewCommand()
		cmd.SetArgs([]string{"--lang", "go"})

		err := cmd.Execute()
		assert.NoError(t, err)

		_, statErr := os.Stat(".golangci.yml")
		assert.NoError(t, statErr)
	})
}

func TestGithubDependabotCommandSuccess(t *testing.T) {
	t.Run("creates config for go language", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cmd := createGithubDependabotCommand()
		cmd.SetArgs([]string{"--lang", "go"})

		err := cmd.Execute()
		assert.NoError(t, err)

		_, statErr := os.Stat(".github/dependabot.yml")
		assert.NoError(t, statErr)
	})
}

func TestCodeGithubReleasePlease(t *testing.T) {
	t.Run("creates all release-please files", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initTestGitRepo(t, "main")

		err := codeGithubReleasePlease()
		assert.NoError(t, err)

		_, statErr := os.Stat(".github/workflows/release-please.yml")
		assert.NoError(t, statErr)

		_, statErr = os.Stat(".github/release-please-config.json")
		assert.NoError(t, statErr)

		_, statErr = os.Stat(".github/release-please-manifest.json")
		assert.NoError(t, statErr)
	})

	t.Run("workflow file starts with generated header", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initTestGitRepo(t, "main")

		err := codeGithubReleasePlease()
		require.NoError(t, err)

		content, readErr := os.ReadFile(".github/workflows/release-please.yml")
		require.NoError(t, readErr)
		assert.True(t, strings.HasPrefix(string(content), "# generated by: yake code github-release-please\n"))
	})

	t.Run("creates directories if not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initTestGitRepo(t, "master")

		err := codeGithubReleasePlease()
		assert.NoError(t, err)

		info, statErr := os.Stat(".github/workflows")
		assert.NoError(t, statErr)
		assert.True(t, info.IsDir())
	})

	t.Run("returns error when no default branch detected", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initTestGitRepo(t, "develop")

		err := codeGithubReleasePlease()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not detect default branch")
	})
}

func TestCreateGithubReleasePleaseCommand(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := createGithubReleasePleaseCommand()

		require.NotNil(t, cmd)
		assert.Equal(t, "github-release-please", cmd.Use)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("has force flag", func(t *testing.T) {
		cmd := createGithubReleasePleaseCommand()

		flag := cmd.Flags().Lookup("force")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("returns error when workflow already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initTestGitRepo(t, "main")
		os.MkdirAll(filepath.Join(".github", "workflows"), 0755)
		os.WriteFile(filepath.Join(".github", "workflows", "release-please.yml"), []byte(""), 0644)

		cmd := createGithubReleasePleaseCommand()
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("force only recreates workflow file", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initTestGitRepo(t, "main")
		os.MkdirAll(filepath.Join(".github", "workflows"), 0755)
		os.WriteFile(filepath.Join(".github", "workflows", "release-please.yml"), []byte("old"), 0644)
		os.WriteFile(filepath.Join(".github", "release-please-config.json"), []byte("existing-config"), 0644)
		os.WriteFile(filepath.Join(".github", "release-please-manifest.json"), []byte("existing-manifest"), 0644)

		cmd := createGithubReleasePleaseCommand()
		cmd.SetArgs([]string{"--force"})

		err := cmd.Execute()
		assert.NoError(t, err)

		workflow, readErr := os.ReadFile(filepath.Join(".github", "workflows", "release-please.yml"))
		require.NoError(t, readErr)
		assert.NotEqual(t, "old", string(workflow))

		config, readErr := os.ReadFile(filepath.Join(".github", "release-please-config.json"))
		require.NoError(t, readErr)
		assert.Equal(t, "existing-config", string(config))

		manifest, readErr := os.ReadFile(filepath.Join(".github", "release-please-manifest.json"))
		require.NoError(t, readErr)
		assert.Equal(t, "existing-manifest", string(manifest))
	})

	t.Run("creates files successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initTestGitRepo(t, "main")

		cmd := createGithubReleasePleaseCommand()
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		assert.NoError(t, err)

		_, statErr := os.Stat(".github/workflows/release-please.yml")
		assert.NoError(t, statErr)
	})
}
