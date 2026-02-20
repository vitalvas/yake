package tools

import (
	"encoding/json"
	"log"
	"os"
)

func WriteJSONFile(filename string, data any) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	if _, err := f.Write(content); err != nil {
		return err
	}

	if _, err := f.WriteString("\n"); err != nil {
		return err
	}

	return nil
}
