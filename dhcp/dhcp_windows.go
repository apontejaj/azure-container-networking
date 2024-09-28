package dhcp

import (
	"context"
	"net"
)

func (c *DHCP) DiscoverRequest(_ context.Context, _ net.HardwareAddr, _ string) error {
	return nil
}
