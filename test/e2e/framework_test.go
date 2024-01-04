package main

import (
	"os"
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

func TestValidateHubbleMetrics(t *testing.T) {
	job := types.NewJob(t)
	defer job.Run()

	user, _ := user.Current()
	testName := user.Username + "-validate-hubble-metrics-17"

	sub := os.Getenv("AZURE_SUBSCRIPTION_ID")
	tenant := os.Getenv("AZURE_TENANT_ID")

	job.AddStep(&azure.CreateResourceGroup{
		SubscriptionID:    sub,
		TenantID:          tenant,
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
		Namespace:     "kube-system",
		LabelSelector: "k8s-app=cilium",
		LocalPort:     "9965",
		RemotePort:    "9965",
	})

	job.AddStep(&hubble.ValidateHubbleMetrics{})

	job.AddStep(&azure.DeleteResourceGroup{})
}
