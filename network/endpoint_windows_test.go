// Copyright 2017 Microsoft. All rights reserved.
// MIT License

//go:build windows
// +build windows

package network

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/iptables"
	"github.com/Azure/azure-container-networking/netio"
	"github.com/Azure/azure-container-networking/netlink"
	"github.com/Azure/azure-container-networking/network/hnswrapper"
	"github.com/Azure/azure-container-networking/platform"
)

var (
	instanceID   = "12345-abcde-789"
	locationPath = "12345-abcde-789-fea14"
	pnpID        = "PCI\\VEN_15B3&DEV_101C&SUBSYS_000715B3&REV_00\\5&8c5acce&0&0"
)

func TestNewAndDeleteEndpointImplHnsV2(t *testing.T) {
	nw := &network{
		Endpoints: map[string]*endpoint{},
	}

	// this hnsv2 variable overwrites the package level variable in network
	// we do this to avoid passing around os specific objects in platform agnostic code
	Hnsv2 = hnswrapper.Hnsv2wrapperwithtimeout{
		Hnsv2: hnswrapper.NewHnsv2wrapperFake(),
	}

	epInfo := &EndpointInfo{
		EndpointID:  "753d3fb6-e9b3-49e2-a109-2acc5dda61f1",
		ContainerID: "545055c2-1462-42c8-b222-e75d0b291632",
		NetNsPath:   "fakeNameSpace",
		IfName:      "eth0",
		Data:        make(map[string]interface{}),
		EndpointDNS: DNSInfo{
			Suffix:  "10.0.0.0",
			Servers: []string{"10.0.0.1, 10.0.0.2"},
			Options: nil,
		},
		MacAddress: net.HardwareAddr("00:00:5e:00:53:01"),
	}
	endpoint, err := nw.newEndpointImplHnsV2(nil, epInfo)
	if err != nil {
		fmt.Printf("+%v", err)
		t.Fatal(err)
	}

	err = nw.deleteEndpointImplHnsV2(endpoint)

	if err != nil {
		fmt.Printf("+%v", err)
		t.Fatal(err)
	}
}

func TestNewEndpointImplHnsv2Timesout(t *testing.T) {
	nw := &network{
		Endpoints: map[string]*endpoint{},
	}

	// this hnsv2 variable overwrites the package level variable in network
	// we do this to avoid passing around os specific objects in platform agnostic code

	hnsFake := hnswrapper.NewHnsv2wrapperFake()

	hnsFake.Delay = 10 * time.Second

	Hnsv2 = hnswrapper.Hnsv2wrapperwithtimeout{
		Hnsv2:          hnsFake,
		HnsCallTimeout: 5 * time.Second,
	}

	epInfo := &EndpointInfo{
		EndpointID:  "753d3fb6-e9b3-49e2-a109-2acc5dda61f1",
		ContainerID: "545055c2-1462-42c8-b222-e75d0b291632",
		NetNsPath:   "fakeNameSpace",
		IfName:      "eth0",
		Data:        make(map[string]interface{}),
		EndpointDNS: DNSInfo{
			Suffix:  "10.0.0.0",
			Servers: []string{"10.0.0.1, 10.0.0.2"},
			Options: nil,
		},
		MacAddress: net.HardwareAddr("00:00:5e:00:53:01"),
	}
	_, err := nw.newEndpointImplHnsV2(nil, epInfo)

	if err == nil {
		t.Fatal("Failed to timeout HNS calls for creating endpoint")
	}
}

func TestDeleteEndpointImplHnsv2Timeout(t *testing.T) {
	nw := &network{
		Endpoints: map[string]*endpoint{},
	}

	Hnsv2 = hnswrapper.NewHnsv2wrapperFake()

	epInfo := &EndpointInfo{
		EndpointID:  "753d3fb6-e9b3-49e2-a109-2acc5dda61f1",
		ContainerID: "545055c2-1462-42c8-b222-e75d0b291632",
		NetNsPath:   "fakeNameSpace",
		IfName:      "eth0",
		Data:        make(map[string]interface{}),
		EndpointDNS: DNSInfo{
			Suffix:  "10.0.0.0",
			Servers: []string{"10.0.0.1, 10.0.0.2"},
			Options: nil,
		},
		MacAddress: net.HardwareAddr("00:00:5e:00:53:01"),
	}
	endpoint, err := nw.newEndpointImplHnsV2(nil, epInfo)
	if err != nil {
		fmt.Printf("+%v", err)
		t.Fatal(err)
	}

	hnsFake := hnswrapper.NewHnsv2wrapperFake()

	hnsFake.Delay = 10 * time.Second

	Hnsv2 = hnswrapper.Hnsv2wrapperwithtimeout{
		Hnsv2:          hnsFake,
		HnsCallTimeout: 5 * time.Second,
	}

	err = nw.deleteEndpointImplHnsV2(endpoint)

	if err == nil {
		t.Fatal("Failed to timeout HNS calls for deleting endpoint")
	}
}

func TestCreateEndpointImplHnsv1Timeout(t *testing.T) {
	nw := &network{
		Endpoints: map[string]*endpoint{},
	}

	hnsFake := hnswrapper.NewHnsv1wrapperFake()

	hnsFake.Delay = 10 * time.Second

	Hnsv1 = hnswrapper.Hnsv1wrapperwithtimeout{
		Hnsv1:          hnsFake,
		HnsCallTimeout: 5 * time.Second,
	}

	epInfo := &EndpointInfo{
		EndpointID:  "753d3fb6-e9b3-49e2-a109-2acc5dda61f1",
		ContainerID: "545055c2-1462-42c8-b222-e75d0b291632",
		NetNsPath:   "fakeNameSpace",
		IfName:      "eth0",
		Data:        make(map[string]interface{}),
		EndpointDNS: DNSInfo{
			Suffix:  "10.0.0.0",
			Servers: []string{"10.0.0.1, 10.0.0.2"},
			Options: nil,
		},
		MacAddress: net.HardwareAddr("00:00:5e:00:53:01"),
	}
	_, err := nw.newEndpointImplHnsV1(epInfo, nil)

	if err == nil {
		t.Fatal("Failed to timeout HNS calls for creating endpoint")
	}
}

func TestDeleteEndpointImplHnsv1Timeout(t *testing.T) {
	nw := &network{
		Endpoints: map[string]*endpoint{},
	}

	Hnsv1 = hnswrapper.NewHnsv1wrapperFake()

	epInfo := &EndpointInfo{
		EndpointID:  "753d3fb6-e9b3-49e2-a109-2acc5dda61f1",
		ContainerID: "545055c2-1462-42c8-b222-e75d0b291632",
		NetNsPath:   "fakeNameSpace",
		IfName:      "eth0",
		Data:        make(map[string]interface{}),
		EndpointDNS: DNSInfo{
			Suffix:  "10.0.0.0",
			Servers: []string{"10.0.0.1, 10.0.0.2"},
			Options: nil,
		},
		MacAddress: net.HardwareAddr("00:00:5e:00:53:01"),
	}
	endpoint, err := nw.newEndpointImplHnsV1(epInfo, nil)
	if err != nil {
		fmt.Printf("+%v", err)
		t.Fatal(err)
	}

	hnsFake := hnswrapper.NewHnsv1wrapperFake()

	hnsFake.Delay = 10 * time.Second

	Hnsv1 = hnswrapper.Hnsv1wrapperwithtimeout{
		Hnsv1:          hnsFake,
		HnsCallTimeout: 5 * time.Second,
	}

	err = nw.deleteEndpointImplHnsV1(endpoint)

	if err == nil {
		t.Fatal("Failed to timeout HNS calls for deleting endpoint")
	}
}

func TestDisableVFDeviceHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	// happy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Disable-PnpDevice") {
			return succededCaseReturn, nil
		}
		return "", nil
	})

	err := DisableVFDevice(instanceID, nm.plClient)
	if err != nil {
		t.Fatal("Failed to test disable VF happy path")
	}
}

func TestDisableVFDeviceUnHappyPathOne(t *testing.T) {
	// set unhappy path
	mockExecClient := platform.NewMockExecClient(true)
	err := DisableVFDevice(instanceID, mockExecClient)
	if err == nil {
		t.Fatal("Failed to test disable VF unhappy path")
	}
}

func TestDisableVFDeviceUnHappyPathTwo(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)
	// set unhappy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		return "", errors.New("Failed to disable VF device")
	})

	err := DisableVFDevice(instanceID, mockExecClient)
	if err == nil {
		t.Fatal("Failed to test disable VF unhappy path")
	}
}

func TestGetLocationPathHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	// happy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Get-PnpDeviceProperty") {
			return succededCaseReturn, nil
		}
		return "", nil
	})

	_, err := GetLocationPath(instanceID, nm.plClient)
	if err != nil {
		t.Fatal("Failed to test get locationPath happy path")
	}
}

func TestGetLocationPathUnHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(true)
	_, err := GetLocationPath(instanceID, mockExecClient)
	if err != nil {
		t.Fatal("Failed to test get locationPath unhappy path")
	}
}

func TestDismountVFDeviceHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	// happy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Dismount-VMHostAssignableDevice") {
			return succededCaseReturn, nil
		}
		return "", nil
	})

	err := DisamountVFDevice(locationPath, nm.plClient)
	if err != nil {
		t.Fatal("Failed to test dismount vf device happy path")
	}
}

func TestDismountVFDeviceUnHappyPathOne(t *testing.T) {
	// set unhappy path
	mockExecClient := platform.NewMockExecClient(true)
	err := DisamountVFDevice(instanceID, mockExecClient)
	if err == nil {
		t.Fatal("Failed to test dismount VF unhappy path")
	}
}

func TestDismountVFDeviceUnHappyPathTwo(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)
	// set unhappy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		return "", errors.New("Failed to dismount VF device")
	})

	err := DisamountVFDevice(instanceID, mockExecClient)
	if err == nil {
		t.Fatal("Failed to test dismount VF unhappy path")
	}
}

func TestGetPnPDeviceIDHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	// happy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Get-PnpDeviceProperty") || strings.Contains(cmd, "Get-VMHostAssignableDevice") {
			return succededCaseReturn, nil
		}

		return "", nil
	})

	_, err := GetPnPDeviceID(instanceID, nm.plClient)
	if err != nil {
		t.Fatal("Failed to test get pnp device id happy path")
	}
}

func TestGetPnPDeviceIDUnHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(true)
	_, err := GetPnPDeviceID(instanceID, mockExecClient)
	if err != nil {
		t.Fatal("Failed to test get pnp device id unhappy path")
	}
}

func TestGetPnPDeviceStateHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	// happy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Get-PnpDevice") {
			return succededCaseReturn, nil
		}

		return "", nil
	})

	_, err := GetPnpDeviceState(instanceID, nm.plClient)
	if err != nil {
		t.Fatal("Failed to test happy path")
	}
}

func TestGetPnPDeviceStateUnHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(true)
	_, err := GetPnpDeviceState(instanceID, mockExecClient)
	if err != nil {
		t.Fatal("Failed to test get pnp device state unhappy path")
	}
}

// endpoint creation is not required for IB
func TestNewEndpointImplHnsv2ForIB(t *testing.T) {
	nw := &network{
		Endpoints: map[string]*endpoint{},
	}

	// this hnsv2 variable overwrites the package level variable in network
	// we do this to avoid passing around os specific objects in platform agnostic code

	hnsFake := hnswrapper.NewHnsv2wrapperFake()

	hnsFake.Delay = 10 * time.Second

	Hnsv2 = hnswrapper.Hnsv2wrapperwithtimeout{
		Hnsv2:          hnsFake,
		HnsCallTimeout: 5 * time.Second,
	}

	epInfo := &EndpointInfo{
		EndpointID: "768e8deb-eth1",
		Data:       make(map[string]interface{}),
		IfName:     "eth1",
		NICType:    cns.BackendNIC,
		PnPID:      pnpID,
	}

	// Happy Path
	endpoint, err := nw.newEndpointImpl(nil, netlink.NewMockNetlink(false, ""), platform.NewMockExecClient(false),
		netio.NewMockNetIO(false, 0), NewMockEndpointClient(nil), NewMockNamespaceClient(), iptables.NewClient(), epInfo)

	if endpoint != nil || err != nil {
		t.Fatal("Endpoint is created for IB")
	}
}
