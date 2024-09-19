package endpointmanager

import (
	"context"
	"net"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/hnsclient"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/restserver"
	"github.com/pkg/errors"
)

type releaseIPsClient interface {
	ReleaseIPs(ctx context.Context, ipconfig cns.IPConfigsRequest) error
	GetEndpoint(ctx context.Context, endpointID string) (*restserver.GetEndpointResponse, error)
}

type EndpointManager struct {
	cli releaseIPsClient
}

func WithPlatformReleaseIPsManager(cli releaseIPsClient) *EndpointManager {
	return &EndpointManager{cli: cli}
}

func (em *EndpointManager) ReleaseIPs(ctx context.Context, ipconfig cns.IPConfigsRequest) error {
	logger.Printf("deleting HNS Endpoint asynchronously")
	// remove HNS endpoint
	if err := em.deleteEndpoint(ctx, ipconfig.InfraContainerID); err != nil {
		logger.Errorf("failed to remove HNS endpoint %s", err.Error())
	}
	em.cli.ReleaseIPs(ctx, ipconfig)
	return nil
}

// GetEndpoint calls the EndpointHandlerAPI in CNS to retrieve the state of a given EndpointID
func (em *EndpointManager) getHNSEndpointID(ctx context.Context, endpointID string) ([]string, [][]net.IPNet, error) {
	endpointResponse, err := em.cli.GetEndpoint(ctx, endpointID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read the endpoint from CNS state")
	}
	var hnsEndpointIDs []string
	var ips [][]net.IPNet
	for _, ipInfo := range endpointResponse.EndpointInfo.IfnameToIPMap {
		hnsEndpointID := ipInfo.HnsEndpointID
		// we need to get the HNSENdpoint via the IP address if the HNSEndpointID is not present in the statefile
		if ipInfo.HnsEndpointID == "" {
			if ipInfo.IPv4 != nil {
				ips = append(ips, ipInfo.IPv4)
			}
			if ipInfo.IPv6 != nil {
				ips = append(ips, ipInfo.IPv6)
			}
		} else {
			hnsEndpointIDs = append(hnsEndpointIDs, hnsEndpointID)
		}
	}
	return hnsEndpointIDs, ips, nil
}

// call GetEndpoint API to get the state and then remove assiciated HNS
func (em *EndpointManager) deleteEndpoint(ctx context.Context, containerid string) error {
	hnsEndpointIDs, ips, err := em.getHNSEndpointID(ctx, containerid)
	if err != nil {
		return errors.Wrap(err, "failed to read the endpoint from CNS state")
	}
	for _, hnsEndpointID := range hnsEndpointIDs {
		logger.Printf("deleting HNS Endpoint with id %v", hnsEndpointID)
		if err := hnsclient.DeleteHNSEndpointbyID(hnsEndpointID); err != nil {
			logger.Errorf("failed to remove HNS endpoint %s %s", hnsEndpointID, err.Error())
		}
	}
	for _, ip := range ips {
		// we need to get the HNSENdpoint via the IP address if the HNSEndpointID is not present in the statefile
		var hnsEndpointID string
		if hnsEndpointID, err = hnsclient.GetHNSEndpointbyIP(ip, ip); err != nil {
			return errors.Wrap(err, "failed to find HNS endpoint with id")
		}
		logger.Printf("deleting HNS Endpoint with id %v", hnsEndpointID)
		if err := hnsclient.DeleteHNSEndpointbyID(hnsEndpointID); err != nil {
			logger.Errorf("failed to remove HNS endpoint %s %s", hnsEndpointID, err.Error())
		}
	}
	return err
}
