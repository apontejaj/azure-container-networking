//go:generate go run generate_yaml.go

package main

import (
	"fmt"
	"os"

	k8s "github.com/Azure/azure-container-networking/test/e2e/framework/kubernetes"
	"gopkg.in/yaml.v2"
)

func main() {
	testresources := "../manifests/"
	GenerateKapingerYAML(testresources)
}

func GenerateKapingerYAML(folder string) {
	kappiefolder := folder + "/kapinger"
	err := os.MkdirAll(kappiefolder, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating folder %s: %v", kappiefolder, err)
		return
	}
	// Create a sample Deployment object
	c := k8s.CreateKapingerDeployment{
		KapingerNamespace: "default",
		KapingerReplicas:  "1",
	}

	resources := map[string]interface{}{
		"kapinger-deployment.yaml":         c.GetKapingerDeployment(),
		"kapinger-service.yaml":            c.GetKapingerService(),
		"kapinger-serviceaccount.yaml":     c.GetKapingerServiceAccount(),
		"kapinger-clusterrole.yaml":        c.GetKapingerClusterRole(),
		"kapinger-clusterrolebinding.yaml": c.GetKapingerClusterRoleBinding(),
	}

	for filename, obj := range resources {
		yamlBytes, err := yaml.Marshal(obj)
		if err != nil {
			fmt.Println("Error marshalling object: ", err)
		}
		err = WriteYAMLToFile(yamlBytes, kappiefolder+"/"+filename)
		if err != nil {
			fmt.Println("Error writing YAML to file: ", err)
		}
	}
}
