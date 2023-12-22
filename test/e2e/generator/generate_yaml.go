//go:generate go run generate_yaml.go

package main

import "github.com/Azure/azure-container-networking/test/e2e/kubernetes"

func main() {
	testresources := "../testresources"
	kubernetes.GenerateKapingerYAML(testresources)
}
