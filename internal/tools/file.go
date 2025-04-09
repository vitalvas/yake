package tools

import (
	"log"
	"os"
)

func WriteStringToFile(fileName string, content string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if _, err := file.WriteString(content); err != nil {
		return err
	}

	return nil
}
