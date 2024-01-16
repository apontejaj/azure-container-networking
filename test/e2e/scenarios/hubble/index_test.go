package hubble

import (
	"os"
	"os/user"
	"strconv"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/test/e2e/framework/azure"
	"github.com/Azure/azure-container-networking/test/e2e/framework/types"
)

const (
	// netObsRGtag is used to tag resources created by this test suite
	netObsRGtag = "-e2e-netobs-"
)

// Objectives
// - Steps are reusable
// - Steps parameters are saved to the context of the job
// - Once written to the job context, the values are immutable
// - Steps have access to the job context and read/write to it
// - Cluster resources used in code should be able to be generated to yaml for easy manual repro
// - Avoid shell/ps calls wherever possible and use go libraries for typed parameters (avoid capturing error codes/stderr/stdout)

func TestDropHubbleMetrics(t *testing.T) {
	job := types.NewJob(t)
	defer job.Run()

	curuser, _ := user.Current()

	testName := curuser.Username + netObsRGtag + strconv.FormatInt(time.Now().Unix(), 10)
	sub := os.Getenv("AZURE_SUBSCRIPTION_ID")

	job.AddStep(&azure.CreateResourceGroup{
		SubscriptionID:    sub,
		ResourceGroupName: testName,
		Location:          "westus2",
	}, nil)

	job.AddStep(&azure.CreateVNet{
		VnetName:         "testvnet",
		VnetAddressSpace: "10.0.0.0/9",
	}, nil)

	job.AddStep(&azure.CreateSubnet{
		SubnetName:         "testsubnet",
		SubnetAddressSpace: "10.0.0.0/12",
	}, nil)

	job.AddStep(&azure.CreateBYOCiliumCluster{
		ClusterName:  testName,
		PodCidr:      "10.128.0.0/9",
		DNSServiceIP: "192.168.0.10",
		ServiceCidr:  "192.168.0.0/28",
	}, nil)

	job.AddStep(&azure.GetAKSKubeConfig{
		KubeConfigFilePath: "./test.pem",
	}, nil)

	job.AddScenario(ValidateDropMetric()...)

	job.AddStep(&azure.DeleteResourceGroup{}, nil)
}
