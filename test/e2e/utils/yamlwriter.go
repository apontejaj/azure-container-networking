package utils

import (
	"fmt"
	"os"
)

func WriteYAMLToFile(yamlBytes []byte, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("YAML error creating file: %s", err)
	}
	defer file.Close()

	_, err = file.Write(yamlBytes)
	if err != nil {
		return fmt.Errorf("YAML error writing to file: %s", err)
	}

	fmt.Printf("YAML written to file: %s\n", filePath)
	return nil
}
