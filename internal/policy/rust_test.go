package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.vitalvas.com/yake/internal/config"
)

func Test_checkRustASCIIOnly(t *testing.T) {
	t.Run("passes with ASCII Rust source", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		code := "fn main() {\n    let greeting = \"hello\";\n}\n"
		require.NoError(t, os.WriteFile("main.rs", []byte(code), 0644))

		err := checkRustASCIIOnly()

		assert.NoError(t, err)
	})

	t.Run("detects non-ASCII characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		code := fmt.Sprintf("fn main() {\n    let greeting = %q;\n}\n", nonASCIIFixtureWord())
		require.NoError(t, os.WriteFile("main.rs", []byte(code), 0644))

		err := checkRustASCIIOnly()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ASCII-only violations")
		assert.Contains(t, err.Error(), "main.rs:2:24")
		assert.Contains(t, err.Error(), "U+00E9")
	})

	t.Run("bypasses line with skip-ascii directive", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		code := fmt.Sprintf("fn main() {\n    let _ = %q; //yake:skip-ascii\n}\n", nonASCIIFixtureWord())
		require.NoError(t, os.WriteFile("main.rs", []byte(code), 0644))

		err := checkRustASCIIOnly()

		assert.NoError(t, err)
	})

	t.Run("directive only bypasses its own line", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		code := fmt.Sprintf("fn main() {\n    let a = %q; //yake:skip-ascii\n    let b = %q;\n}\n",
			nonASCIIFixtureWord(), nonASCIIFixtureWord())
		require.NoError(t, os.WriteFile("main.rs", []byte(code), 0644))

		err := checkRustASCIIOnly()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "main.rs:3:")
		assert.NotContains(t, err.Error(), "main.rs:2:")
	})

	t.Run("skips tests directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.MkdirAll("tests", 0755))
		code := fmt.Sprintf("fn it_works() {\n    let _ = %q;\n}\n", nonASCIIFixtureWord())
		require.NoError(t, os.WriteFile("tests/integration.rs", []byte(code), 0644))

		err := checkRustASCIIOnly()

		assert.NoError(t, err)
	})

	t.Run("ignores non-Rust files", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		code := fmt.Sprintf("let greeting = %q\n", nonASCIIFixtureWord())
		require.NoError(t, os.WriteFile("notes.txt", []byte(code), 0644))

		err := checkRustASCIIOnly()

		assert.NoError(t, err)
	})
}

func Test_checkCargoTomlNoComment(t *testing.T) {
	t.Run("passes with no comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("Cargo.toml", []byte("[package]\nname = \"x\"\n"), 0644))

		err := checkCargoTomlNoComment()

		assert.NoError(t, err)
	})

	t.Run("passes when Cargo.toml is absent", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		err := checkCargoTomlNoComment()

		assert.NoError(t, err)
	})

	t.Run("detects full-line comment", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("Cargo.toml", []byte("[package]\n# a note\nname = \"x\"\n"), 0644))

		err := checkCargoTomlNoComment()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cargo.toml:2:1")
	})

	t.Run("detects trailing comment", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("Cargo.toml", []byte("[package]\nname = \"x\" # note\n"), 0644))

		err := checkCargoTomlNoComment()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cargo.toml:2:12")
	})

	t.Run("detects comment in a sub-crate Cargo.toml", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("Cargo.toml", []byte("[workspace]\n"), 0644))
		require.NoError(t, os.MkdirAll("crates/app", 0755))
		require.NoError(t, os.WriteFile("crates/app/Cargo.toml", []byte("[package]\n# note\nname = \"app\"\n"), 0644))

		err := checkCargoTomlNoComment()

		require.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("%s:2:1", filepath.Join("crates", "app", "Cargo.toml")))
	})

	t.Run("ignores hash inside a string value", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("Cargo.toml", []byte("[package]\ncolor = \"#fff\"\nname = 'a#b'\n"), 0644))

		err := checkCargoTomlNoComment()

		assert.NoError(t, err)
	})
}

func TestRunRustChecks(t *testing.T) {
	t.Run("passes with ASCII Rust source", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("main.rs", []byte("fn main() {}\n"), 0644))

		err := RunRustChecks()
		assert.NoError(t, err)
	})

	t.Run("fails on non-ASCII source", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		code := fmt.Sprintf("fn main() {\n    let _ = %q;\n}\n", nonASCIIFixtureWord())
		require.NoError(t, os.WriteFile("main.rs", []byte(code), 0644))

		err := RunRustChecks()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ASCII-only violations")
	})

	t.Run("skips disabled check", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cfgContent := "rust_policy:\n  ascii_only:\n    enable: false\n"
		require.NoError(t, os.WriteFile(config.File, []byte(cfgContent), 0644))

		code := fmt.Sprintf("fn main() {\n    let _ = %q;\n}\n", nonASCIIFixtureWord())
		require.NoError(t, os.WriteFile("main.rs", []byte(code), 0644))

		err := RunRustChecks()
		assert.NoError(t, err)
	})
}
