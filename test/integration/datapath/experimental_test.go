//go:build connection

package connection

import (
	"context"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/test/internal/kubernetes"

	"github.com/stretchr/testify/require"
)

var (
	goldpingerSelector = flag.String("app", "goldpinger", "App goldpinger for test pods")
	podVnetKey         = "customervnet"
)

func TestOrchestration(t *testing.T) {
	ctx := context.Background()

	// Create a node to node info map
	nodes, err := kubernetes.GetNodeListByLabelSelector(ctx, clientset, ncNodeLabelSelector)
	require.NoError(t, err, fmt.Sprintf("Listing nodes with label selector %s failed.", ncNodeLabelSelector))
	t.Logf("Nodes found: %v", len(nodes.Items))
	nodeNameToNodeInfo = make(map[string]*nodeInfo)
	for _, node := range nodes.Items {
		nodeInfo := &nodeInfo{}
		// Get the node's ip
		for _, address := range node.Status.Addresses {
			if address.Type == nodeAddressType {
				nodeInfo.ip = address.Address
			}
		}

		// Set desired NC count
		nodeInfo.desiredNCCount = testConfig.DesiredNCsPerNode
		nodeNameToNodeInfo[node.ObjectMeta.Name] = nodeInfo
	}
	cleanupFN := buildCleanupFunc(ctx, t)
	defer t.Cleanup(cleanupFN)

	// hard code populating of NC information created during running NA
	podCounter := 0
	customerVnets := 2
	nodeNames := []string{"aks-podpool-15091151-vmss000004", "aks-podpool-15091151-vmss000005"}

	for k := 0; k < customerVnets; k++ {
		for _, nodeName := range nodeNames {
			t.Logf("Selected node %v for vnet %v", nodeName, k)
			nodeInfo := nodeNameToNodeInfo[nodeName]
			for i := 0; i < nodeInfo.desiredNCCount; i++ {
				nc := ncInfo{
					PodName:      fmt.Sprint(podPrefix, podCounter),
					PodNamespace: namespace,
					NCID:         "unused",
				}
				nodeInfo.allocatedNCs = append(nodeInfo.allocatedNCs, nc)
				podCounter += 1
			}
		}
	}
	t.Log("successfully populated NC information into nodeInfo")

	t.Log("---------------- SWIFT TESTS -----------------------")
	// Label and make pods
	customervnet := "testingValue"
	for nodeName, nodeInfo := range nodeNameToNodeInfo {
		for _, nc := range nodeInfo.allocatedNCs {
			pod := kubernetes.MustParsePod(testConfig.GoldpingerPodYamlPath)

			pod.Spec.NodeSelector = make(map[string]string)
			pod.Spec.NodeSelector[nodeLabelKey] = nodeName
			pod.ObjectMeta.Labels[podVnetKey] = customervnet
			pod.Name = nc.PodName
			pod.Namespace = nc.PodNamespace
			kubernetes.MustCreatePod(ctx, clientset.CoreV1().Pods(pod.Namespace), pod)
			require.NoError(t, err, fmt.Sprintf("Deploying Pod: %s failed with error: %v", pod.Name, err))
			t.Logf("Successfully deployed pod: %s", pod.Name)
		}
	}
	t.Log("successfully created customer pods")

	// podLabelSelector := k8sutils.CreateLabelSelector(podLabelKey, goldpingerSelector)

	// t.Run("Linux ping tests", func(t *testing.T) {
	// 	// Check goldpinger health
	// 	t.Run("all pods have IPs assigned", func(t *testing.T) {
	// 		ipCheckTimeout := defaultTimeoutSeconds * time.Second
	// 		ipCheckCtx, cancel := context.WithTimeout(ctx, ipCheckTimeout)
	// 		defer cancel()

	// 		select {
	// 		case <-ipCheckCtx.Done():
	// 			t.Fatalf("pod ips could not be assigned in %d seconds: %v", ipCheckTimeout, ctx.Err())
	// 		default:
	// 			err := k8sutils.WaitForPodsRunning(ctx, clientset, *podNamespace, podLabelSelector)
	// 			if err != nil {
	// 				t.Fatalf("Pods are not in running state due to %+v", err)
	// 			}
	// 		}

	// 		t.Log("all pods have been allocated IPs")
	// 	})
	// })
	time.Sleep(60 * time.Second)

	// 	t.Run("all linux pods can ping each other", func(t *testing.T) {
	// 		clusterCheckCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	// 		defer cancel()

	// 		pfOpts := k8s.PortForwardingOpts{
	// 			Namespace:     *podNamespace,
	// 			LabelSelector: podLabelSelector,
	// 			LocalPort:     9090,
	// 			DestPort:      8080,
	// 		}

	// 		pf, err := k8s.NewPortForwarder(restConfig, t, pfOpts)
	// 		if err != nil {
	// 			t.Fatal(err)
	// 		}

	// 		portForwardCtx, cancel := context.WithTimeout(ctx, defaultTimeoutSeconds*time.Second)
	// 		defer cancel()

	// 		portForwardFn := func() error {
	// 			err := pf.Forward(portForwardCtx)
	// 			if err != nil {
	// 				t.Logf("unable to start port forward: %v", err)
	// 				return err
	// 			}
	// 			return nil
	// 		}

	// 		if err := defaultRetrier.Do(portForwardCtx, portForwardFn); err != nil {
	// 			t.Fatalf("could not start port forward within %d: %v", defaultTimeoutSeconds, err)
	// 		}
	// 		defer pf.Stop()

	// 		gpClient := goldpinger.Client{Host: pf.Address()}
	// 		clusterCheckFn := func() error {
	// 			clusterState, err := gpClient.CheckAll(clusterCheckCtx)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			stats := goldpinger.ClusterStats(clusterState)
	// 			stats.PrintStats()
	// 			if stats.AllPingsHealthy() {
	// 				return nil
	// 			}

	// 			return errors.New("not all pings are healthy")
	// 		}
	// 		retrier := retry.Retrier{Attempts: goldpingerRetryCount, Delay: goldpingerDelayTimeSeconds * time.Second}
	// 		if err := retrier.Do(clusterCheckCtx, clusterCheckFn); err != nil {
	// 			t.Fatalf("goldpinger pods network health could not reach healthy state after %d seconds: %v", goldpingerRetryCount*goldpingerDelayTimeSeconds, err)
	// 		}

	// 		t.Log("all pings successful!")
	// 	})
	// })

}

func buildCleanupFunc(ctx context.Context, t *testing.T) func() {
	return func() {
		t.Log("---------------- CLEANUP -----------------------")
		// Delete pods
		t.Log("Deleting pods...")
		for _, nodeInfo := range nodeNameToNodeInfo {
			for _, nc := range nodeInfo.allocatedNCs {
				kubernetes.MustDeletePod(ctx, clientset.CoreV1().Pods(nc.PodNamespace), nc.PodName)
			}
		}

		t.Log("Finished deleting pods")
	}
}

// func createPods(t *testing.T, ctx context.Context, customerVnet string) {
// 	// Label and make pods
// 	/*
// 		    number of nodes
// 			names of vnets
// 			pod names & namespaces
// 	*/
// 	for nodeName, nodeInfo := range nodeNameToNodeInfo {
// 		for _, vnet := range nodeInfo.allocatedVnets {
// 			for customerVnet, nc := range vnet {
// 				pod, err := k8sutils.MustParsePod(testConfig.GoldpingerPodYamlPath)
// 				require.NoError(t, err, "Parsing pod deployment failed")
// 				pod.Spec.NodeSelector = make(map[string]string)
// 				pod.Spec.NodeSelector[nodeLabelKey] = nodeName
// 				pod.ObjectMeta.Labels[podVnetKey] = customerVnet
// 				pod.Name = nc.PodName
// 				pod.Namespace = nc.PodNamespace
// 				err = k8sutils.MustCreateOrUpdatePod(ctx, clientset.CoreV1().Pods(pod.Namespace), pod)
// 				require.NoError(t, err, "Creating pods failed")
// 				require.NoError(t, err, fmt.Sprintf("Deploying Pod: %s failed with error: %v", pod.Name, err))
// 				t.Logf("Successfully deployed pod: %s", pod.Name)
// 			}
// 		}
// 	}
// 	t.Log("successfully created customer pods")
// }
