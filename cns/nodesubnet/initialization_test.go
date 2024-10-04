package nodesubnet_test

import (
	"context"
	"net"
	"testing"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/cnireconciler"
	"github.com/Azure/azure-container-networking/cns/ipam"
	"github.com/Azure/azure-container-networking/cns/nodesubnet"
	"github.com/Azure/azure-container-networking/cns/restserver"
	"github.com/Azure/azure-container-networking/cns/types"
	"github.com/Azure/azure-container-networking/crd/nodenetworkconfig/api/v1alpha"
	"github.com/Azure/azure-container-networking/store"
)

func getMockStore() store.KeyValueStore {
	mockStore := store.NewMockStore("")
	endpointState := map[string]*restserver.EndpointInfo{
		"12e65d89e58cb23c784e97840cf76866bfc9902089bdc8e87e9f64032e312b0b": {
			PodName:      "coredns-54b69f46b8-ldmwr",
			PodNamespace: "kube-system",
			IfnameToIPMap: map[string]*restserver.IPInfo{
				"eth0": {
					IPv4: []net.IPNet{
						{
							IP:   net.IPv4(10, 10, 0, 52),
							Mask: net.CIDRMask(24, 32),
						},
					},
				},
			},
		},
		"1fc5176913a3a1a7facfb823dde3b4ded404041134fef4f4a0c8bba140fc0413": {
			PodName:      "load-test-7f7d49687d-wxc9p",
			PodNamespace: "load-test",
			IfnameToIPMap: map[string]*restserver.IPInfo{
				"eth0": {
					IPv4: []net.IPNet{
						{
							IP:   net.IPv4(10, 10, 0, 63),
							Mask: net.CIDRMask(24, 32),
						},
					},
				},
			},
		},
	}

	err := mockStore.Write(restserver.EndpointStoreKey, endpointState)
	if err != nil {
		return nil
	}
	return mockStore
}

type MockIpamStateReconciler struct{}

func (m *MockIpamStateReconciler) ReconcileIPAMState(ncRequests []*cns.CreateNetworkContainerRequest, podInfoByIP map[string]cns.PodInfo, nnc *v1alpha.NodeNetworkConfig) types.ResponseCode {
	if len(ncRequests) == 1 && len(ncRequests[0].SecondaryIPConfigs) == len(podInfoByIP) {
		return types.Success
	}

	return types.UnexpectedError
}

func TestNewCNSPodInfoProvider(t *testing.T) {
	tests := []struct {
		name       string
		store      store.KeyValueStore
		want       map[string]cns.PodInfo
		wantErr    bool
		reconciler ipam.IpamStateReconciler
		exp        int
	}{
		{
			name:  "good",
			store: getMockStore(),
			want: map[string]cns.PodInfo{
				"10.10.0.52": cns.NewPodInfo("12e65d89e58cb23c784e97840cf76866bfc9902089bdc8e87e9f64032e312b0b", "12e65d89e58cb23c784e97840cf76866bfc9902089bdc8e87e9f64032e312b0b", "coredns-54b69f46b8-ldmwr", "kube-system"),
				"10.10.0.63": cns.NewPodInfo("1fc5176913a3a1a7facfb823dde3b4ded404041134fef4f4a0c8bba140fc0413", "1fc5176913a3a1a7facfb823dde3b4ded404041134fef4f4a0c8bba140fc0413", "load-test-7f7d49687d-wxc9p", "load-test"),
			},
			wantErr:    false,
			reconciler: &MockIpamStateReconciler{},
			exp:        2,
		},
	}

	for _, tt := range tests {
		tt := tt
		ctx, cancel := testContext(t)
		defer cancel()

		t.Run(tt.name, func(t *testing.T) {
			podInfoByIPProvider, err := cnireconciler.NewCNSPodInfoProvider(tt.store)
			checkErr(t, err, false)

			got, err := nodesubnet.ReconcileInitialCNSState(ctx, tt.reconciler, podInfoByIPProvider)
			checkErr(t, err, tt.wantErr)
			if got != tt.exp {
				t.Errorf("got %d IPs reconciled, expected %d", got, tt.exp)
			}
		})
	}
}

// testContext creates a context from the provided testing.T that will be
// canceled if the test suite is terminated.
func testContext(t *testing.T) (context.Context, context.CancelFunc) {
	if deadline, ok := t.Deadline(); ok {
		return context.WithDeadline(context.Background(), deadline)
	}
	return context.WithCancel(context.Background())
}
