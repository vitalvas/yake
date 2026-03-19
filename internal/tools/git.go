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

type GitHubRepo struct {
	Owner string
	Name  string
}

func DetectGitHubRepo() (GitHubRepo, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return GitHubRepo{}, fmt.Errorf("could not detect GitHub repository: no origin remote")
	}

	url := strings.TrimSpace(string(out))

	owner, name, ok := parseGitHubURL(url)
	if !ok {
		return GitHubRepo{}, fmt.Errorf("could not parse GitHub repository from URL: %s", url)
	}

	return GitHubRepo{Owner: owner, Name: name}, nil
}

func parseGitHubURL(url string) (owner, name string, ok bool) {
	url = strings.TrimSuffix(url, ".git")

	if path, found := strings.CutPrefix(url, "https://github.com/"); found {
		parts := strings.Split(path, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], true
		}
	}

	if path, found := strings.CutPrefix(url, "git@github.com:"); found {
		parts := strings.Split(path, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], true
		}
	}

	return "", "", false
}
