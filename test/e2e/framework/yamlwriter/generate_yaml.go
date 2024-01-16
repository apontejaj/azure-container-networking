//go:generate go run generate_yaml.go

package main

func main() {
	testresources := "../../manifests/"
	GenerateKapingerYAML(testresources)
}
