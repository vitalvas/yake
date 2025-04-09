package tools

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func WriteYamlFile(filename string, data interface{}) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)

	if err := enc.Encode(data); err != nil {
		return err
	}

	return nil
}
