package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"
)

type CreateCluster struct {
	SubscriptionID    string
	ResourceGroupName string
	Location          string
	ClusterName       string
}

func (c *CreateCluster) Run() error {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	poller, err := clientFactory.NewManagedClustersClient().BeginCreateOrUpdate(ctx, c.ResourceGroupName, c.ClusterName, GetStarterClusterTemplate(c.Location), nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}

	return nil
}

func GetStarterClusterTemplate(location string) armcontainerservice.ManagedCluster {
	id := armcontainerservice.ResourceIdentityTypeSystemAssigned
	return armcontainerservice.ManagedCluster{
		Location: to.Ptr(location),
		Tags: map[string]*string{
			"archv2": to.Ptr(""),
			"tier":   to.Ptr("production"),
		},
		Properties: &armcontainerservice.ManagedClusterProperties{
			AddonProfiles: map[string]*armcontainerservice.ManagedClusterAddonProfile{},
			AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{
				{
					Type:               to.Ptr(armcontainerservice.AgentPoolTypeVirtualMachineScaleSets),
					AvailabilityZones:  []*string{to.Ptr("1")},
					Count:              to.Ptr[int32](3),
					EnableNodePublicIP: to.Ptr(false),
					Mode:               to.Ptr(armcontainerservice.AgentPoolModeSystem),
					OSType:             to.Ptr(armcontainerservice.OSTypeLinux),
					ScaleDownMode:      to.Ptr(armcontainerservice.ScaleDownModeDelete),
					VMSize:             to.Ptr("Standard_D4s_v3"),
					Name:               to.Ptr("nodepool1"),
					MaxPods:            to.Ptr(int32(250)),
				}},
			KubernetesVersion:       to.Ptr(""),
			DNSPrefix:               to.Ptr("dnsprefix1"),
			EnablePodSecurityPolicy: to.Ptr(false),
			EnableRBAC:              to.Ptr(true),
			LinuxProfile:            nil,
			NetworkProfile: &armcontainerservice.NetworkProfile{
				LoadBalancerSKU: to.Ptr(armcontainerservice.LoadBalancerSKUStandard),
				OutboundType:    to.Ptr(armcontainerservice.OutboundTypeLoadBalancer),
				NetworkPlugin:   to.Ptr(armcontainerservice.NetworkPluginAzure),
			},
			WindowsProfile: &armcontainerservice.ManagedClusterWindowsProfile{
				AdminPassword: to.Ptr("replacePassword1234$"),
				AdminUsername: to.Ptr("azureuser"),
			},
		},
		Identity: &armcontainerservice.ManagedClusterIdentity{
			Type: &id,
		},

		SKU: &armcontainerservice.ManagedClusterSKU{
			Name: to.Ptr(armcontainerservice.ManagedClusterSKUName("Base")),
			Tier: to.Ptr(armcontainerservice.ManagedClusterSKUTierStandard),
		},
	}
}

func (c *CreateCluster) ExpectError() bool {
	return false
}

func (c *CreateCluster) SaveParametersToJob() bool {
	return true
}

func (c *CreateCluster) Prevalidate() error {
	return nil
}

func (c *CreateCluster) Postvalidate() error {
	return nil
}
