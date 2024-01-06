package tools

import (
	"os"

	"gopkg.in/yaml.v3"
)

func WriteYamlFile(filename string, data interface{}) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)

	if err := enc.Encode(data); err != nil {
		return err
	}

	return nil
}
