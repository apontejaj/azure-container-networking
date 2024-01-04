package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

type DeleteResourceGroup struct {
	SubscriptionID    string
	ResourceGroupName string
	Location          string
}

func (d *DeleteResourceGroup) Run() error {
	log.Printf("deleting resource group %s...", d.ResourceGroupName)
	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armresources.NewClientFactory(d.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v\n", err)
	}

	poller, err := clientFactory.NewResourceGroupsClient().BeginDelete(ctx, d.ResourceGroupName, &armresources.ResourceGroupsClientBeginDeleteOptions{ForceDeletionTypes: to.Ptr("Microsoft.Compute/virtualMachines,Microsoft.Compute/virtualMachineScaleSets")})
	if err != nil {
		log.Fatalf("failed to finish the request: %v\n", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to poll delete rg: %v\n", err)
	}

	log.Printf("resource group %s deleted successfully", d.ResourceGroupName)
	return nil
}

func (d *DeleteResourceGroup) Prevalidate() error {
	return nil
}

func (d *DeleteResourceGroup) ExpectError() bool {
	return false
}

func (d *DeleteResourceGroup) SaveParametersToJob() bool {
	return true
}

func (d *DeleteResourceGroup) Postvalidate() error {
	return nil
}
