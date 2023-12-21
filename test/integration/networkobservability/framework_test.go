package main

import (
	"testing"

	"github.com/Azure/azure-container-networking/test/integration/networkobservability/azure"
	"github.com/Azure/azure-container-networking/test/integration/networkobservability/types"
)

// Child defines the child struct with Requires and Produces methods

func TestJobs(t *testing.T) {
	job := types.NewJob()

	// when adding the step, we pass the variable to the local job context
	job.AddStep(&azure.CreateResourceGroup{
		SubscriptionID:    "123",
		ResourceGroupName: "test",
		Location:          "westus",
	})

	job.AddStep(&azure.CreateCluster{
		ClusterName: "test",
	})

	job.AddStep(&azure.GetKubeConfig{
		KubeConfigFilePath: "./test.yaml",
	})

	// Go through each step, and make sure that the values are set
	// for each step, if the value is not passed, then the step can pull
	// from the job context
	// i.e. if we create a resource group in one step,
	// then that will be set in the context and a create cluster later on will be able to pull that value
	job.Validate()
}
