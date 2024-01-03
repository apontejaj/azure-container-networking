package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-container-networking/test/e2e/types"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
)

type CreateVNet struct {
	SubscriptionID    string
	ResourceGroupName string
	Location          string
	VnetName          string
	VnetAddressSpace  string
}

func (c *CreateVNet) Run() error {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armnetwork.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	log.Printf("creating vnet %s in resource group %s...", c.VnetName, c.ResourceGroupName)

	poller, err := clientFactory.NewVirtualNetworksClient().BeginCreateOrUpdate(ctx, c.ResourceGroupName, c.VnetName, armnetwork.VirtualNetwork{
		Location: to.Ptr(c.Location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{
					to.Ptr(c.VnetAddressSpace)},
			},
			FlowTimeoutInMinutes: to.Ptr[int32](10),
		},
	}, nil)

	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}
	return nil
}

func (c *CreateVNet) ExpectError() bool {
	return false
}

func (c *CreateVNet) SaveParametersToJob() bool {
	return true
}

func (c *CreateVNet) Prevalidate() error {
	return nil
}

func (c *CreateVNet) Postvalidate() error {
	return nil
}

type CreateSubnet struct {
	SubscriptionID     string
	ResourceGroupName  string
	Location           string
	VnetName           string
	SubnetName         string
	SubnetAddressSpace string
}

func (c *CreateSubnet) Run(values *types.JobValues) error {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armnetwork.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	log.Printf("creating subnet %s in resource group %s...", c.SubnetName, c.ResourceGroupName)

	poller, err := clientFactory.NewSubnetsClient().BeginCreateOrUpdate(ctx, c.ResourceGroupName, c.VnetName, c.SubnetName, armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: to.Ptr(c.SubnetAddressSpace),
		},
	}, nil)

	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}
	return nil
}

func (c *CreateSubnet) ExpectError() bool {
	return false
}

func (c *CreateSubnet) SaveParametersToJob() bool {
	return true
}

func (c *CreateSubnet) Prevalidate(values *types.JobValues) error {
	return nil
}

func (c *CreateSubnet) Postvalidate(values *types.JobValues) error {
	return nil
}
