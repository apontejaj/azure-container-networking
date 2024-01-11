package main

import (
	"os"
	"os/user"
	"strconv"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/test/e2e/azure"
	"github.com/Azure/azure-container-networking/test/e2e/hubble"
	k8s "github.com/Azure/azure-container-networking/test/e2e/kubernetes"
	"github.com/Azure/azure-container-networking/test/e2e/types"
)

const (
	netObsRGtag = "-e2e-netobs-"
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

	curuser, _ := user.Current()

	testName := curuser.Username + netObsRGtag + strconv.FormatInt(time.Now().Unix(), 10)

	sub := os.Getenv("AZURE_SUBSCRIPTION_ID")

	job.AddStep(&azure.CreateResourceGroup{
		SubscriptionID:    sub,
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
		// ClusterName: "matmerr-e2e-netobs-1704927259",
		// ResourceGroupName:  "matmerr-e2e-netobs-1704927259",
		// Location:           "westus2",
		// SubscriptionID:     "d9eabe18-12f6-4421-934a-d7e2327585f5",
		KubeConfigFilePath: "./test.pem",
	})

	job.AddStep(&k8s.CreateKapingerDeployment{
		KapingerNamespace: "kube-system",
		KapingerReplicas:  "1",
	})

	job.AddStep(&k8s.CreateAgnhostStatefulSet{
		AgnhostName:      "agnhost-a",
		AgnhostNamespace: "kube-system",
	})

	job.AddStep(&k8s.ExecInPod{
		PodName:      "agnhost-a-0",
		PodNamespace: "kube-system",
		Command:      "curl -s google.com",
	})

	job.AddStep(&k8s.PortForward{
		Namespace:             "kube-system",
		LabelSelector:         "k8s-app=cilium",
		LocalPort:             "9965",
		RemotePort:            "9965",
		OptionalLabelAffinity: "app=agnhost-a", // port forward to a pod on a node that also has this pod with this label, assuming same namespace
	})

	job.AddStep(&types.Sleep{
		Duration: 15 * time.Second,
	})

	job.AddStep(&hubble.ValidateHubbleMetrics{})

	job.AddStep(&azure.DeleteResourceGroup{})
}

func TestCreateAMAWorkspace(t *testing.T) {
	job := types.NewJob(t)
	defer job.Run()

	curuser, _ := user.Current()
	testName := curuser.Username + netObsRGtag + strconv.FormatInt(time.Now().Unix(), 10)

	sub := os.Getenv("AZURE_SUBSCRIPTION_ID")

	job.AddStep(&azure.CreateAzureMonitor{
		SubscriptionID:    sub,
		ResourceGroupName: testName,
		ClusterName:       testName,
		Location:          "westus2",
	})

	job.AddStep(&azure.CreateAzureMonitor{})
}

func TestDNSTraffic(t *testing.T) {
	job := types.NewJob(t)
	defer job.Run()

	curuser, _ := user.Current()
	testName := curuser.Username + netObsRGtag + strconv.FormatInt(time.Now().Unix(), 10)

	sub := os.Getenv("AZURE_SUBSCRIPTION_ID")

	job.AddStep(&azure.CreateAzureMonitor{
		SubscriptionID:    sub,
		ResourceGroupName: testName,
		ClusterName:       testName,
		Location:          "westus2",
	})

	job.AddStep(&azure.CreateAzureMonitor{})
}
