package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

type CreateResourceGroup struct {
	SubscriptionID    string
	TenantID          string
	ResourceGroupName string
	Location          string
}

func (c *CreateResourceGroup) Run() error {
	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armresources.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	log.Println("resource group\"" + c.ResourceGroupName + "\" creating...")

	_, err = clientFactory.NewResourceGroupsClient().CreateOrUpdate(ctx, c.ResourceGroupName, armresources.ResourceGroup{
		Location: to.Ptr(c.Location),
	}, nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}

	log.Println("resource group \"" + c.ResourceGroupName + "\" created successfully")
	return nil
}

func (c *CreateResourceGroup) Prevalidate() error {
	return nil
}

func (c *CreateResourceGroup) ExpectError() bool {
	return false
}

func (c *CreateResourceGroup) SaveParametersToJob() bool {
	return true
}

func (c *CreateResourceGroup) Postvalidate() error {
	return nil
}
