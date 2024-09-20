package restserver

import (
	"context"
	"net/netip"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/logger"
	nodesubnet "github.com/Azure/azure-container-networking/cns/nodesubnet"
	"github.com/Azure/azure-container-networking/cns/types"
	errors "github.com/pkg/errors"
)

var _ nodesubnet.IPConsumer = &HTTPRestService{}

// Implement the UpdateIPsForNodeSubnet method for HTTPRestService
func (service *HTTPRestService) UpdateIPsForNodeSubnet(primaryIP netip.Addr, secondaryIPs []netip.Addr) error {
	networkContainerRequest, err := nodesubnet.CreateNodeSubnetNCRequest(primaryIP, secondaryIPs)
	if err != nil {
		return errors.Wrap(err, "creating network container request")
	}

	code, msg := service.saveNetworkContainerGoalState(*networkContainerRequest)
	if code == types.NodeSubnetSecondaryIPChange {
		logger.Debugf("Secondary IP change detected, updating fetch interval")
		service.nodesubnetIPFetcher.UpdateFetchIntervalForObservedDiff()
	} else if code != types.Success {
		logger.Debugf("Error in processing IP change, refresh interval not updated")
		return errors.Errorf("failed to save fetched ips. code: %d, message %s", code, msg)
	} else {
		logger.Debugf("No secondary IP change detected, updating fetch interval")
		service.nodesubnetIPFetcher.UpdateFetchIntervalForNoObservedDiff()
	}

	// saved NC successfully, generate conflist to indicate CNS is ready
	go service.MustGenerateCNIConflistOnce()
	return nil
}

func (service *HTTPRestService) InitializeNodeSubnet(ctx context.Context) {
	// Set orchestrator type
	orchestrator := cns.SetOrchestratorTypeRequest{
		OrchestratorType: cns.KubernetesNodeSubnet,
	}
	service.SetNodeOrchestrator(&orchestrator)
	service.nodesubnetIPFetcher = nodesubnet.NewIPFetcher(service.nma, service, 0, 0)
	service.nodesubnetIPFetcher.Start(ctx)
}
