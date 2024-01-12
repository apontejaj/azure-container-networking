package hubble

import (
	"time"

	k8s "github.com/Azure/azure-container-networking/test/e2e/framework/kubernetes"
	"github.com/Azure/azure-container-networking/test/e2e/framework/types"
	"github.com/Azure/azure-container-networking/test/e2e/hubble/scenarios"
)

const (
	// Hubble drop reasons
	UnsupportedL3Protocol = "UNSUPPORTED_L3_PROTOCOL"
	PolicyDenied          = "POLICY_DENIED"

	// L4 protocols
	TCP = "TCP"
	UDP = "UDP"

	Delay = 30 * time.Second
)

func HubbleDropScenario() []types.StepWrapper {
	return []types.StepWrapper{
		{
			Step: &k8s.CreateKapingerDeployment{
				KapingerNamespace: "kube-system",
				KapingerReplicas:  "1",
			},
		},
		{
			Step: &k8s.CreateDenyAllNetworkPolicy{
				NetworkPolicyNamespace: "kube-system",
				DenyAllLabelSelector:   "app=agnhost-a",
			},
		},
		{
			Step: &k8s.CreateAgnhostStatefulSet{
				AgnhostName:      "agnhost-a",
				AgnhostNamespace: "kube-system",
			},
		},
		{
			Step: &k8s.ExecInPod{
				PodName:      "agnhost-a-0",
				PodNamespace: "kube-system",
				Command:      "curl -s -m 5 bing.com",
			},
			Opts: &types.StepOptions{
				ExpectError: true,
			},
		},
		{ // metrics take some time to show up, so sleep for a bit
			Step: &types.Sleep{
				Duration: Delay,
			},
			Opts: &types.StepOptions{
				SaveParametersToJob: false,
			},
		},
		{
			Step: &k8s.PortForward{
				Namespace:             "kube-system",
				LabelSelector:         "k8s-app=cilium",
				LocalPort:             "9965",
				RemotePort:            "9965",
				OptionalLabelAffinity: "app=agnhost-a", // port forward to a pod on a node that also has this pod with this label, assuming same namespace
			},
		},
		{
			Step: &scenarios.ValidateHubbleDropMetric{
				PortForwardedHubblePort: "9965",
				Source:                  "agnhost-a",
				Reason:                  PolicyDenied,
				Protocol:                UDP,
			},
		},
	}
}
