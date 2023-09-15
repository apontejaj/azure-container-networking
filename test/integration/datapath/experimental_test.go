//go:build connection

package connection

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	rbacCleanUpFn = func(t *testing.T) {}
)

func TestOrchestration(t *testing.T) {
	ctx := context.Background()

	t.Log("---------------- SWIFT TESTS -----------------------")

	// Label and make pods
	for nodeName, nodeInfo := range nodeNameToNodeInfo {
		for _, nc := range nodeInfo.allocatedNCs {
			pod, err := k8s.ParsePod(testConfig.GoldpingerPodYamlPath)
			require.NoError(t, err, "Parsing pod deployment failed")
			pod.Spec.NodeSelector = make(map[string]string)
			pod.Spec.NodeSelector[nodeLabelKey] = nodeName
			pod.Name = nc.PodName
			pod.Namespace = nc.PodNamespace
			err = k8sShim.CreateOrUpdatePod(ctx, pod.Namespace, pod)
			require.NoError(t, err, "Creating pods failed")
			err = checkIfPodIsReady(namespace, pod.Name, defaultRetrier)
			require.NoError(t, err, fmt.Sprintf("Deploying Pod: %s failed with error: %v", pod.Name, err))
			t.Logf("Successfully deployed pod: %s", pod.Name)
		}
	}
	t.Log("successfully created customer pods")

}

func buildCleanupFunc(ctx context.Context, dncDeployment v1.Deployment, cnsDaemonset, cniManagerDaemonset, hostDaemonset v1.DaemonSet, dncConfigMap, cnsConfigMap apiv1.ConfigMap, goldpingerPod apiv1.Pod, t *testing.T) func() {
	return func() {
		t.Log("---------------- CLEANUP -----------------------")
		// Delete pods
		t.Log("Deleting pods...")
		for _, nodeInfo := range nodeNameToNodeInfo {
			for _, nc := range nodeInfo.allocatedNCs {
				if err := k8sShim.DeletePod(ctx, nc.PodNamespace, nc.PodName); err != nil {
					clearnupErrors.Collect(err)
				}
			}
		}

		require.NoError(t, clearnupErrors.Any(), "Failed to delete either pods, ncs, subnet, vnet or dnc nodes, dnc, cns, cni manager or host daemonset with error: %+v", clearnupErrors.GetErrors())
		t.Log("Successfully deleted pods, ncs, unjoined subnet, vnet, deleted dnc node, dnc, cns, cni and host daemonset")
	}
}
