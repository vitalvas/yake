package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

func DetectDefaultBranch() (string, error) {
	out, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		branch := strings.TrimSpace(string(out))
		branch = strings.TrimPrefix(branch, "refs/remotes/origin/")

		if branch != "" {
			return branch, nil
		}
	}

	if err := exec.Command("git", "rev-parse", "--verify", "refs/heads/main").Run(); err == nil {
		return "main", nil
	}

	if err := exec.Command("git", "rev-parse", "--verify", "refs/heads/master").Run(); err == nil {
		return "master", nil
	}

	out, err = exec.Command("git", "symbolic-ref", "--short", "HEAD").Output()
	if err == nil {
		branch := strings.TrimSpace(string(out))
		if branch == "main" || branch == "master" {
			return branch, nil
		}
	}

	return "", fmt.Errorf("could not detect default branch")
}
