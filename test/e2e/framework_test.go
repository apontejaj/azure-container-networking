package main

import (
	"os/user"
	"testing"

	"github.com/Azure/azure-container-networking/test/e2e/azure"
	"github.com/Azure/azure-container-networking/test/e2e/hubble"
	k8s "github.com/Azure/azure-container-networking/test/e2e/kubernetes"
	"github.com/Azure/azure-container-networking/test/e2e/types"
)

// Objectives
// - Steps are reusable
// - Steps parameters are saved to the context of the job
// - Once written to the job context, the values are immutable
// - Steps have access to the job context and read/write to it
// - Cluster resources used in code should be able to be generated to yaml for easy manual repro
// - Avoid shell/ps calls wherever possible and use go libraries for typed parameters (avoid capturing error codes/stderr/stdout)

func TestCreateCluster(t *testing.T) {
	job := types.NewJob(t)
	defer job.Run()

	job.AddStep(&azure.CreateResourceGroup{
		SubscriptionID:    "9b8218f9-902a-4d20-a65c-e98acec5362f",
		ResourceGroupName: "matmerr-e2e-framework-test2",
		Location:          "westus2",
	})

	job.AddStep(&azure.CreateCluster{
		ClusterName: "matmerr-e2e-framework-test",
		//ResourceGroupName: "matmerr-e2e-framework-zdfgtest2",
	})

	job.AddStep(&azure.GetAKSKubeConfig{
		KubeConfigFilePath: "./test.yaml",
	})

	job.AddStep(&k8s.CreateKapingerDeployment{
		KapingerNamespace: "kapinger",
		KapingerReplicas:  "1",
	})

	//job.AddStep(&azure.DeleteCluster{})
}

func TestAddJobs(t *testing.T) {
	job := types.NewJob(t)
	defer job.Run()

	job.AddStep(&azure.CreateResourceGroup{
		SubscriptionID: "9b8218f9-902a-4d20-a65c-e98acec5362f",
		Location:       "westus2",
	})
}

func TestDeployKapinger(t *testing.T) {
	job := types.NewJob(t)
	defer job.Run()

	job.AddStep(&k8s.CreateKapingerDeployment{
		KapingerNamespace:  "default",
		KapingerReplicas:   "1",
		KubeConfigFilePath: "./test.yaml",
	})
}

func TestPortForward(t *testing.T) {
	job := types.NewJob(t)
	defer job.Run()

	user, _ := user.Current()
	testName := user.Name + " validate-hubble-metrics"

	job.AddStep(&azure.CreateResourceGroup{
		SubscriptionID:    "9b8218f9-902a-4d20-a65c-e98acec5362f",
		ResourceGroupName: testName,
		Location:          "westus2",
	})

	job.AddStep(&azure.CreateVNet{
		VnetName:         "testvnet",
		VnetAddressSpace: "10.0.0.0/9",
	})

	job.AddStep(&azure.CreateSubnet{
		SubnetName:         "testsubnet",
		SubnetAddressSpace: "10.0.0.0/12",
	})

	job.AddStep(&azure.CreateBYOCiliumCluster{
		ClusterName:  testName,
		PodCidr:      "10.128.0.0/9",
		DNSServiceIP: "192.168.0.10",
		ServiceCidr:  "192.168.0.0/28",
	})

	job.AddStep(&azure.GetAKSKubeConfig{
		KubeConfigFilePath: "./test.pem",
	})

	job.AddStep(&k8s.CreateKapingerDeployment{
		KapingerNamespace: "default",
		KapingerReplicas:  "1",
	})

	job.AddStep(&k8s.PortForward{
		KubeConfigFilePath: "./test.pem",
		Namespace:          "kube-system",
		LabelSelector:      "k8s-app=cilium",
		LocalPort:          "9965",
		RemotePort:         "9965",
	})

	job.AddStep(&hubble.ValidateHubbleMetrics{})
}
