package kubernetes

import (
	"errors"
	"fmt"
	"os"

	"github.com/Azure/azure-container-networking/test/e2e/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ApplyDeploymentFromFile struct {
	DeploymentFilename string
}

func getConfig(configpath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configpath)
	if err != nil {
		fmt.Println("Error building kubeconfig: ", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("Error creating Kubernetes client: ", err)
		return nil, err
	}
	return clientset, err
}

func (c *ApplyDeploymentFromFile) Run(values *types.JobValues) error {
	_, err := getConfig(c.DeploymentFilename)
	if err != nil {
		fmt.Println("Error getting config: ", err)
		return err
	}

	return nil
}

func (c *ApplyDeploymentFromFile) ExpectError() bool {
	return false
}

func (c *ApplyDeploymentFromFile) SaveParametersToJob() bool {
	return false
}

func (c *ApplyDeploymentFromFile) Prevalidate(values *types.JobValues) error {

	if _, err := os.Stat(c.DeploymentFilename); errors.Is(err, os.ErrNotExist) {
		// path/to/whatever does not exist
	}

	return nil
}
