package github

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name        string               `yaml:"name"`
	On          WorkflowOn           `yaml:"on"`
	Concurrency *WorkflowConcurrency `yaml:"concurrency,omitempty"`
	Permissions WorkflowPermissions  `yaml:"permissions,omitempty"`
	Jobs        OrderedJobs          `yaml:"jobs"`
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
	WorkflowDispatch *struct{}        `yaml:"workflow_dispatch,omitempty"`
	MergeGroup       *WorkflowTrigger `yaml:"merge_group,omitempty"`
	PullRequest      *WorkflowTrigger `yaml:"pull_request,omitempty"`
	Push             *WorkflowTrigger `yaml:"push,omitempty"`
}

type WorkflowTrigger struct {
	Branches []string `yaml:"branches,omitempty"`
	Types    []string `yaml:"types,omitempty"`
	Paths    []string `yaml:"paths,omitempty"`
}

type WorkflowConcurrency struct {
	Group            string `yaml:"group,omitempty"`
	CancelInProgress string `yaml:"cancel-in-progress,omitempty"`
}

type WorkflowPermissions struct {
	Contents     string `yaml:"contents,omitempty"`
	Issues       string `yaml:"issues,omitempty"`
	PullRequests string `yaml:"pull-requests,omitempty"`
}

type OrderedJobs []JobEntry

type JobEntry struct {
	Name string
	Job  WorkflowJob
}

func (o OrderedJobs) Get(name string) (WorkflowJob, bool) {
	for _, e := range o {
		if e.Name == name {
			return e.Job, true
		}
	}

	return WorkflowJob{}, false
}

func (o OrderedJobs) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}

	for _, e := range o {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: e.Name}

		var valNode yaml.Node
		if err := valNode.Encode(e.Job); err != nil {
			return nil, err
		}

		node.Content = append(node.Content, keyNode, &valNode)
	}

	return node, nil
}

type WorkflowJob struct {
	Name    string            `yaml:"name,omitempty"`
	RunsOn  string            `yaml:"runs-on"`
	Needs   []string          `yaml:"needs,omitempty"`
	If      string            `yaml:"if,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	Outputs map[string]string `yaml:"outputs,omitempty"`
	Steps   []WorkflowStep    `yaml:"steps"`
}

type WorkflowStep struct {
	Name string            `yaml:"name,omitempty"`
	If   string            `yaml:"if,omitempty"`
	Uses string            `yaml:"uses,omitempty"`
	ID   string            `yaml:"id,omitempty"`
	Run  string            `yaml:"run,omitempty"`
	Env  map[string]string `yaml:"env,omitempty"`
	With map[string]string `yaml:"with,omitempty"`
}
