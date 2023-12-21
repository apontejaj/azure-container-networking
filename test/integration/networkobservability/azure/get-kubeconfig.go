package azure

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-container-networking/test/integration/networkobservability/types"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"
)

type GetKubeConfig struct {
	ClusterName        string
	SubscriptionID     string
	ResourceGroup      string
	Location           string
	KubeConfigFilePath string
}

func (c *GetKubeConfig) Run(values *types.JobValues) error {

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	res, err := clientFactory.NewManagedClustersClient().GetAccessProfile(ctx, c.ResourceGroup, c.ClusterName, "clusterUser", nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}

	fmt.Println(res.ManagedClusterAccessProfile.Properties.KubeConfig)
	return nil
}

func (c *GetKubeConfig) Prevalidate(values *types.JobValues) error {
	return nil
}

func (c *GetKubeConfig) DryRun(values *types.JobValues) error {
	return nil
}
