package azure

import (
	"context"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"
)

type GetAKSKubeConfig struct {
	ClusterName        string
	SubscriptionID     string
	ResourceGroupName  string
	Location           string
	KubeConfigFilePath string
}

func (c *GetAKSKubeConfig) Run() error {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	res, err := clientFactory.NewManagedClustersClient().ListClusterUserCredentials(ctx, c.ResourceGroupName, c.ClusterName, nil)

	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}

	err = os.WriteFile(c.KubeConfigFilePath, []byte(res.Kubeconfigs[0].Value), 0644)
	log.Printf("kubeconfig for cluster %s in resource group %s written to %s\n", c.ClusterName, c.ResourceGroupName, c.KubeConfigFilePath)
	return nil
}

func (c *GetAKSKubeConfig) ExpectError() bool {
	return false
}

func (c *GetAKSKubeConfig) SaveParametersToJob() bool {
	return true
}

func (c *GetAKSKubeConfig) Prevalidate() error {
	return nil
}

func (c *GetAKSKubeConfig) Postvalidate() error {
	return nil
}
