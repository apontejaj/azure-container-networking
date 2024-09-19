package endpointmanager

import (
	"context"

	"github.com/Azure/azure-container-networking/cns"
)

type releaseIPsClient interface {
	ReleaseIPs(ctx context.Context, ipconfig cns.IPConfigsRequest) error
}

// start to move the abstraction on these platform specific things
// NewEndpointManager is noop for Linux
func WithPlatformReleaseIPsManager(cli releaseIPsClient) releaseIPsClient {
	return cli
}
