// Copyright 2017 Microsoft. All rights reserved.
// MIT License

//go:build windows
// +build windows

package network

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/network/hnswrapper"
	"github.com/Azure/azure-container-networking/platform"
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

func TestNewEndpointImplHnsV2ForBackendNICHappyPath(t *testing.T) {
	pnpID := "PCI\\VEN_15B3&DEV_101C&SUBSYS_000715B3&REV_00\\5&8c5acce&0&0"

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
		IfName:      "eth2",
		Data:        make(map[string]interface{}),
		EndpointDNS: DNSInfo{
			Suffix:  "10.0.0.0",
			Servers: []string{"10.0.0.1, 10.0.0.2"},
			Options: nil,
		},
		MacAddress: net.HardwareAddr("00:00:5e:00:53:01"),
		NICType:    cns.BackendNIC,
		PnPID:      pnpID,
	}

	// happy path
	// should return nil if nicType is BackendNIC
	endpoint, err := nw.newEndpointImplHnsV2(nil, epInfo)
	if endpoint != nil && err != nil {
		t.Fatal("HNS Endpoint is created with BackendNIC")
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
		t.Fatal("Failed to test happy path")
	}
}

func TestDisableVFDeviceUnHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	// happy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Disable-PnpDevice") {
			return failedCaseReturn, errTestFailure
		}
		return "", nil
	})

	err := DisableVFDevice(instanceID, nm.plClient)
	if err == nil {
		t.Fatal("Failed to test unhappy path with failing to execute Disable-PnpDevice command")
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
		t.Fatal("Failed to test happy path")
	}
}

func TestGetLocationPathUnHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	// happy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Get-PnpDeviceProperty") {
			return failedCaseReturn, errTestFailure
		}
		return "", nil
	})

	_, err := GetLocationPath(instanceID, nm.plClient)
	if err == nil {
		t.Fatal("Failed to test unhappy path with failing to execute Get-PnpDeviceProperty command")
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
		t.Fatal("Failed to test happy path")
	}
}

func TestDismountVFDeviceUnHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Dismount-VMHostAssignableDevice") {
			return failedCaseReturn, errTestFailure
		}
		return "", nil
	})

	err := DisamountVFDevice(locationPath, nm.plClient)
	if err == nil {
		t.Fatal("Failed to test unhappy path with failing to execute Dismount-VMHostAssignableDevice command")
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
		t.Fatal("Failed to test happy path")
	}
}

func TestGetPnPDeviceIDUnHappyPath(t *testing.T) {
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Get-PnpDeviceProperty") {
			return succededCaseReturn, nil
		}

		// fail secondary command execution
		if strings.Contains(cmd, "Get-VMHostAssignableDevice") {
			return failedCaseReturn, errTestFailure
		}

		return "", nil
	})

	_, err := GetPnPDeviceID(instanceID, nm.plClient)
	if err == nil {
		t.Fatal("Failed to test unhappy path with failing to Get PnpDevice ID command")
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
	mockExecClient := platform.NewMockExecClient(false)

	nm := &networkManager{
		plClient: mockExecClient,
	}

	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		// fail command execution
		if strings.Contains(cmd, "Get-PnpDevice") {
			return failedCaseReturn, errTestFailure
		}

		return "", nil
	})

	_, err := GetPnpDeviceState(instanceID, nm.plClient)
	if err == nil {
		t.Fatal("Failed to test unhappy path with failing to Get PnpDevice state command")
	}
}
