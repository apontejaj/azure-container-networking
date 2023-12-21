package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-container-networking/test/integration/networkobservability/types"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"
)

type DeleteCluster struct {
	ClusterName       string
	SubscriptionID    string
	ResourceGroupName string
	Location          string
}

func (c *DeleteCluster) Run(values *types.JobValues) error {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	res, err := clientFactory.NewManagedClustersClient().Get(ctx, c.ResourceGroupName, c.ClusterName, nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	// You could use response here. We use blank identifier for just demo purposes.
	_ = res
	return nil
}

func (c *DeleteCluster) Prevalidate(values *types.JobValues) error {
	return nil
}

func (c *DeleteCluster) DryRun(values *types.JobValues) error {
	return nil
}
