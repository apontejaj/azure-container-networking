package utils

import (
	"fmt"
	"os"
)

func WriteYAMLToFile(yamlBytes []byte, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(yamlBytes)
	if err != nil {
		return err
	}

	fmt.Printf("YAML written to file: %s\n", filePath)
	return nil
}
