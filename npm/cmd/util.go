package main

import (
	"context"
	"fmt"

	"github.com/Azure/azure-container-networking/npm/metrics"
	"github.com/Azure/azure-container-networking/npm/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func labelNode(clientset *kubernetes.Clientset, nodeName, labelValue string) error {
	msg := fmt.Sprintf("labeling this node %s with %s=%s", nodeName, util.NPMNodeLabelKey, labelValue)
	metrics.SendLog(util.NpmID, msg, metrics.PrintLog)

	k8sNode, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get k8s node. nodeName: %s. err: %w", nodeName, err)
	}

	if k8sNode.Labels == nil {
		k8sNode.Labels = make(map[string]string)
	}

	if val, ok := k8sNode.Labels[util.NPMNodeLabelKey]; ok && val == labelValue {
		klog.Infof("node %s already labeled with %s=%s", nodeName, util.NPMNodeLabelKey, labelValue)
		return nil
	}

	k8sNode.Labels[util.NPMNodeLabelKey] = labelValue

	_, err = clientset.CoreV1().Nodes().Update(context.TODO(), k8sNode, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update k8s node. nodeName: %s. err: %w", nodeName, err)
	}

	return nil
}
