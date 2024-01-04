package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"
)

type DeleteCluster struct {
	ClusterName       string
	SubscriptionID    string
	ResourceGroupName string
	Location          string
}

func (d *DeleteCluster) Run() error {
	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(d.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	log.Printf("deleting cluster %s in resource group %s...", d.ClusterName, d.ResourceGroupName)
	poller, err := clientFactory.NewManagedClustersClient().BeginDelete(ctx, d.ResourceGroupName, d.ClusterName, nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}
	return nil
}

func (d *DeleteCluster) ExpectError() bool {
	return false
}

func (d *DeleteCluster) SaveParametersToJob() bool {
	return true
}

func (d *DeleteCluster) Prevalidate() error {
	return nil
}

func (d *DeleteCluster) Postvalidate() error {
	return nil
}
