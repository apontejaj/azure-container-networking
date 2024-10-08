package nodesubnet

import (
	"context"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/ipam"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/types"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
)

func ReconcileInitialCNSState(_ context.Context, ipamReconciler ipam.StateReconciler, podInfoByIPProvider cns.PodInfoByIPProvider) (int, error) {
	// Get previous PodInfo state from podInfoByIPProvider
	podInfoByIP, err := podInfoByIPProvider.PodInfoByIP()
	if err != nil {
		return 0, errors.Wrap(err, "provider failed to provide PodInfoByIP")
	}

	logger.Printf("Reconciling initial CNS state with %d IPs", len(podInfoByIP))

	// Create a network container request that holds all the IPs from PodInfoByIP
	secondaryIPs := maps.Keys(podInfoByIP)
	ncRequest := CreateNodeSubnetNCRequest(secondaryIPs)
	responseCode := ipamReconciler.ReconcileIPAMState([]*cns.CreateNetworkContainerRequest{ncRequest}, podInfoByIP, nil)

	if responseCode != types.Success {
		return 0, errors.Errorf("failed to reconcile initial CNS state: %d", responseCode)
	}

	return len(secondaryIPs), nil
}
