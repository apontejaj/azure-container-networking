package dhcp

import (
	"context"
	"net"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"
)

const (
	dummyIPAddressStr    = "169.254.128.10"
	dummySubnetMask      = "255.255.128.0"
	addIPAddressDelay    = 4 * time.Second
	deleteIPAddressDelay = 2 * time.Second

	socketTimeoutMillis = 1000
)

var (
	dummyIPAddress = net.IPv4(169, 254, 128, 10) // nolint
	// matches if the string fully consists of zero or more alphanumeric, dots, dashes, parentheses, spaces, or underscores
	allowedInput = regexp.MustCompile(`^[a-zA-Z0-9._\-\(\) ]*$`)
)

type Socket struct {
	fd       windows.Handle
	destAddr windows.SockaddrInet4
}

func NewSocket(destAddr windows.SockaddrInet4) (*Socket, error) {
	// Create a raw socket using windows.WSASocket
	fd, err := windows.WSASocket(windows.AF_INET, windows.SOCK_RAW, windows.IPPROTO_UDP, nil, 0, windows.WSA_FLAG_OVERLAPPED)
	ret := &Socket{
		fd:       fd,
		destAddr: destAddr,
	}
	if err != nil {
		return ret, errors.Wrap(err, "error creating socket")
	}

	// Set IP_HDRINCL to indicate that we are including our own IP header
	err = windows.SetsockoptInt(fd, windows.IPPROTO_IP, windows.IP_HDRINCL, 1)
	if err != nil {
		return ret, errors.Wrap(err, "error setting IP_HDRINCL")
	}
	// Set the SO_BROADCAST option or else we get an error saying that we access a socket in a way forbidden by its access perms
	err = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_BROADCAST, 1)
	if err != nil {
		return ret, errors.Wrap(err, "error setting SO_BROADCAST")
	}
	// Set timeout
	if err = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_RCVTIMEO, socketTimeoutMillis); err != nil {
		return ret, errors.Wrap(err, "error setting receive timeout")
	}
	return ret, nil
}

func (s *Socket) Write(packetBytes []byte) (int, error) {
	err := windows.Sendto(s.fd, packetBytes, 0, &s.destAddr)
	if err != nil {
		return 0, errors.Wrap(err, "failed windows send to")
	}
	return len(packetBytes), nil
}

func (s *Socket) Read(p []byte) (n int, err error) {
	n, _, innerErr := windows.Recvfrom(s.fd, p, 0)
	if innerErr != nil {
		return 0, errors.Wrap(err, "failed windows recv from")
	}
	return n, nil
}

func (s *Socket) Close() error {
	// do not attempt to close invalid fd (happens on socket creation failure)
	if s.fd == windows.InvalidHandle {
		return nil
	}
	// Ensure the file descriptor is closed when done
	if err := windows.Closesocket(s.fd); err != nil {
		return errors.Wrap(err, "error closing dhcp windows socket")
	}
	return nil
}

// issues a dhcp discover request on an interface by assigning an ip to that interface
// then, sends a packet with that interface's dummy ip, and then unassigns the dummy ip
func (c *DHCP) DiscoverRequest(ctx context.Context, macAddress net.HardwareAddr, ifName string) error {
	// validate interface name
	if !allowedInput.MatchString(ifName) {
		return errors.New("invalid dhcp discover request interface name")
	}
	// delete dummy ip off the interface if it already exists
	ret, err := c.execClient.ExecuteCommand(ctx, "netsh", "interface", "ipv4", "delete", "address", ifName, dummyIPAddressStr)
	if err != nil {
		c.logger.Info("Could not remove dummy ip", zap.String("output", ret), zap.Error(err))
	}
	time.Sleep(deleteIPAddressDelay)

	// create dummy ip so we can direct the packet to the correct interface
	ret, err = c.execClient.ExecuteCommand(ctx, "netsh", "interface", "ipv4", "add", "address", ifName, dummyIPAddressStr, dummySubnetMask)
	if err != nil {
		return errors.Wrap(err, "failed to add dummy ip to interface: "+ret)
	}
	// ensure we always remove the dummy ip we added from the interface
	defer func() {
		ret, cleanupErr := c.execClient.ExecuteCommand(ctx, "netsh", "interface", "ipv4", "delete", "address", ifName, dummyIPAddressStr)
		if cleanupErr != nil {
			c.logger.Info("Could not remove dummy ip on leaving function", zap.String("output", ret), zap.Error(err))
		}
	}()
	// it takes time for the address to be assigned
	time.Sleep(addIPAddressDelay)

	// now begin the dhcp request
	txid, err := GenerateTransactionID()
	if err != nil {
		return errors.Wrap(err, "failed to generate transaction id")
	}

	// Prepare an IP and UDP header
	raddr := &net.UDPAddr{IP: net.IPv4bcast, Port: dhcpServerPort}
	laddr := &net.UDPAddr{IP: dummyIPAddress, Port: dhcpClientPort}

	dhcpDiscover, err := buildDHCPDiscover(macAddress, txid)
	if err != nil {
		return errors.Wrap(err, "failed to build dhcp discover")
	}

	// Fill out the headers, add payload, and construct the full packet
	bytesToSend, err := MakeRawUDPPacket(dhcpDiscover, *raddr, *laddr)
	if err != nil {
		return errors.Wrap(err, "failed to make raw udp packet")
	}

	destAddr := windows.SockaddrInet4{
		Addr: [4]byte{255, 255, 255, 255}, // Destination IP
		Port: dhcpServerPort,              // Destination Port
	}
	// create new socket for writing and reading
	sock, err := NewSocket(destAddr)
	defer func() {
		// always clean up the socket, even if we fail while setting options
		closeErr := sock.Close()
		if closeErr != nil {
			c.logger.Error("Error closing dhcp socket:", zap.Error(closeErr))
		}
	}()
	if err != nil {
		return errors.Wrap(err, "failed to create socket")
	}

	_, err = sock.Write(bytesToSend)
	if err != nil {
		return errors.Wrap(err, "failed to write to dhcp socket")
	}

	c.logger.Info("DHCP Discover packet was sent successfully", zap.Any("transactionID", txid))

	// Wait for DHCP response (Offer)
	err = c.receiveDHCPResponse(ctx, sock, txid)
	if err != nil {
		return errors.Wrap(err, "failed to read from dhcp socket")
	}

	return nil
}
