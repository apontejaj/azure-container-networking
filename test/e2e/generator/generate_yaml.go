//go:generate go run generate_yaml.go

package main

import k8s "github.com/Azure/azure-container-networking/test/e2e/kubernetes"

func main() {
	testresources := "../manifests/"
	k8s.GenerateKapingerYAML(testresources)
}
