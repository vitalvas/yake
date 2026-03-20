package githook

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	conventionalCommitRe = regexp.MustCompile(`^(feat|fix|perf|deps|revert|docs|chore|style|refactor|test|build|ci)(\(.+\))?!?: `)
	deniedTypes          = map[string]bool{
		"style":    true,
		"refactor": true,
		"test":     true,
		"build":    true,
		"ci":       true,
	}
)

func RunCommitMsg(msgFile string) error {
	data, err := os.ReadFile(msgFile)
	if err != nil {
		return fmt.Errorf("reading commit message file: %w", err)
	}

	if lockData, err := os.ReadFile(".git/lock_commit"); err == nil {
		if msg := strings.TrimSpace(string(lockData)); msg != "" {
			return fmt.Errorf("%s", msg)
		}

		return fmt.Errorf("commit is locked")
	}

	msg := strings.TrimRight(string(data), "\n")

	if strings.Contains(msg, "\n") {
		return fmt.Errorf("commit message must be a single line")
	}

	if strings.HasPrefix(msg, "Merge ") {
		return nil
	}

	if !conventionalCommitRe.MatchString(msg) {
		return nil
	}

	if strings.Contains(msg, "!:") || strings.Contains(msg, "! :") {
		return fmt.Errorf("breaking change indicator (!) is not allowed")
	}

	commitType := strings.SplitN(msg, "(", 2)[0]
	commitType = strings.SplitN(commitType, ":", 2)[0]
	commitType = strings.TrimSuffix(commitType, "!")

	if deniedTypes[commitType] {
		return fmt.Errorf("commit type '%s' is not allowed, use one of: feat, fix, perf, deps, revert, docs, chore", commitType)
	}

	return nil
}
