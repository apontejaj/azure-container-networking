package azure

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-container-networking/test/e2e/types"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

type CreateResourceGroup struct {
	SubscriptionID    string
	ResourceGroupName string
	Location          string
}

func (c *CreateResourceGroup) Run(values *types.JobValues) error {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armresources.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	fmt.Println("resource group\"" + c.ResourceGroupName + "\" creating...")

	_, err = clientFactory.NewResourceGroupsClient().CreateOrUpdate(ctx, c.ResourceGroupName, armresources.ResourceGroup{
		Location: to.Ptr(c.Location),
	}, nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}

	fmt.Println("resource group\"" + c.ResourceGroupName + "\" created successfully")
	return nil
}

func (c *CreateResourceGroup) Prevalidate(values *types.JobValues) error {
	return nil
}

func (c *CreateResourceGroup) ExpectError() bool {
	return false
}

func (c *CreateResourceGroup) SaveParametersToJob() bool {
	return true
}
