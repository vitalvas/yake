package tools

import (
	"bufio"
	"bytes"
	"os"
	"strings"
)

const gitignoreFile = ".gitignore"

// EnsureGitignoreEntries makes sure every entry is present in .gitignore. It
// reads the existing file (treating a missing file as empty), appends only the
// entries that are not already present as exact lines, and writes the file back,
// creating it when it does not exist. Entries already present are left
// untouched and the file is not modified when nothing needs to be added.
func EnsureGitignoreEntries(entries ...string) error {
	data, err := os.ReadFile(gitignoreFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	existing := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		existing[strings.TrimSpace(scanner.Text())] = true
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	var missing []string
	for _, entry := range entries {
		if !existing[entry] {
			missing = append(missing, entry)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.Write(data)

	if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
		buf.WriteByte('\n')
	}

	for _, entry := range missing {
		buf.WriteString(entry)
		buf.WriteByte('\n')
	}

	return os.WriteFile(gitignoreFile, buf.Bytes(), 0644)
}
