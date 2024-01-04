package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"
)

type DeleteCluster struct {
	ClusterName       string
	SubscriptionID    string
	TenantID          string
	ResourceGroupName string
	Location          string
}

func (c *DeleteCluster) Run() error {
	cred, err := azidentity.NewDefaultAzureCredential(to.Ptr(azidentity.DefaultAzureCredentialOptions{
		TenantID: c.TenantID,
	}))
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	log.Printf("deleting cluster %s in resource group %s...", c.ClusterName, c.ResourceGroupName)
	poller, err := clientFactory.NewManagedClustersClient().BeginDelete(ctx, c.ResourceGroupName, c.ClusterName, nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}
	return nil
}

func (c *DeleteCluster) ExpectError() bool {
	return false
}

func (c *DeleteCluster) SaveParametersToJob() bool {
	return true
}

func (c *DeleteCluster) Prevalidate() error {
	return nil
}

func (c *DeleteCluster) Postvalidate() error {
	return nil
}
