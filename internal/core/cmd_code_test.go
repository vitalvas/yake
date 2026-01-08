package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		require.GreaterOrEqual(t, len(codeSubcommands), 3)

		uses := make([]string, len(codeSubcommands))
		for i, cmd := range codeSubcommands {
			uses[i] = cmd.Use
		}

		assert.Contains(t, uses, "defaults")
		assert.Contains(t, uses, "linter-new")
		assert.Contains(t, uses, "github-dependabot")
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
