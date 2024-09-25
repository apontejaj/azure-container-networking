package nodesubnet

import (
	"context"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/ipam"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
)

func ReconcileInitialCNSState(ctx context.Context, ipamReconciler ipam.IpamStateReconciler, podInfoByIPProvider cns.PodInfoByIPProvider) error {
	// Get previous PodInfo state from podInfoByIPProvider
	podInfoByIP, err := podInfoByIPProvider.PodInfoByIP()
	if err != nil {
		return errors.Wrap(err, "provider failed to provide PodInfoByIP")
	}

	// Create a network container request that holds all the IPs from PodInfoByIP
	secondaryIPs := maps.Keys(podInfoByIP)
	ncRequest, err := CreateNodeSubnetNCRequest(secondaryIPs)
	if err != nil {
		return errors.Wrap(err, "ncRequest creation failed")
	}

	ipamReconciler.ReconcileIPAMState([]*cns.CreateNetworkContainerRequest{ncRequest}, podInfoByIP, nil)
	return nil
}
