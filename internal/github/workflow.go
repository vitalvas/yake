package github

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name        string                 `yaml:"name"`
	On          WorkflowOn             `yaml:"on"`
	Permissions WorkflowPermissions    `yaml:"permissions,omitempty"`
	Jobs        map[string]WorkflowJob `yaml:"jobs"`
}

// Marshal encodes the workflow to YAML with unquoted "on" key.
// yaml.v3 quotes "on" because it is a YAML 1.1 boolean keyword,
// but GitHub Actions requires the unquoted form.
func (w Workflow) Marshal() ([]byte, error) {
	var buf bytes.Buffer

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(w); err != nil {
		return nil, err
	}

	return bytes.Replace(buf.Bytes(), []byte(`"on":`), []byte("on:"), 1), nil
}

type WorkflowOn struct {
	Push        *WorkflowTrigger `yaml:"push,omitempty"`
	PullRequest *WorkflowTrigger `yaml:"pull_request,omitempty"`
}

type WorkflowTrigger struct {
	Branches []string `yaml:"branches,omitempty"`
}

type WorkflowPermissions struct {
	Contents     string `yaml:"contents,omitempty"`
	Issues       string `yaml:"issues,omitempty"`
	PullRequests string `yaml:"pull-requests,omitempty"`
}

type WorkflowJob struct {
	Name    string            `yaml:"name"`
	RunsOn  string            `yaml:"runs-on"`
	Outputs map[string]string `yaml:"outputs,omitempty"`
	Steps   []WorkflowStep    `yaml:"steps"`
}

type WorkflowStep struct {
	Name string            `yaml:"name,omitempty"`
	Uses string            `yaml:"uses,omitempty"`
	ID   string            `yaml:"id,omitempty"`
	Run  string            `yaml:"run,omitempty"`
	With map[string]string `yaml:"with,omitempty"`
}
