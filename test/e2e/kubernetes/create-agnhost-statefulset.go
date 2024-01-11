package k8s

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	AgnhostHTTPPort = 80
	AgnhostReplicas = 1
)

type CreateAgnhostStatefulSet struct {
	AgnhostName        string
	AgnhostNamespace   string
	KubeConfigFilePath string
}

func (c *CreateAgnhostStatefulSet) Run() error {
	config, err := clientcmd.BuildConfigFromFlags("", c.KubeConfigFilePath)
	if err != nil {
		return fmt.Errorf("error building kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resources := []runtime.Object{
		c.getAgnhostDeployment(),
	}

	for i := range resources {
		err = CreateResource(ctx, resources[i], clientset)
		if err != nil {
			return fmt.Errorf("error agnhost component: %w", err)
		}
	}

	err = WaitForPodReady(ctx, clientset, c.AgnhostNamespace, c.AgnhostName)
	if err != nil {
		return fmt.Errorf("error waiting for agnhost pod to be ready: %w", err)
	}

	return nil
}

func (c *CreateAgnhostStatefulSet) ExpectError() bool {
	return false
}

// we'll potentially create multiple deployments, so don't save parameters to the job
func (c *CreateAgnhostStatefulSet) SaveParametersToJob() bool {
	return false
}

func (c *CreateAgnhostStatefulSet) Prevalidate() error {
	return nil
}

func (c *CreateAgnhostStatefulSet) Postvalidate() error {
	return nil
}

func (c *CreateAgnhostStatefulSet) getAgnhostDeployment() *appsv1.StatefulSet {
	reps := int32(AgnhostReplicas)

	return &appsv1.StatefulSet{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      c.AgnhostName,
			Namespace: c.AgnhostNamespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &reps,
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{
					"app":     c.AgnhostName,
					"k8s-app": "agnhost",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"app":     c.AgnhostName,
						"k8s-app": "agnhost",
					},
					Annotations: map[string]string{
						"policy.cilium.io/proxy-visibility": "<Egress/53/UDP/DNS>",
					},
				},

				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							// prefer an even spread across the cluster to avoid scheduling on the same node
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
								{
									Weight: MaxAffinityWeight,
									PodAffinityTerm: v1.PodAffinityTerm{
										TopologyKey: "kubernetes.io/hostname",
										LabelSelector: &metaV1.LabelSelector{
											MatchLabels: map[string]string{
												"k8s-app": "agnhost",
											},
										},
									},
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  c.AgnhostName,
							Image: "k8s.gcr.io/e2e-test-images/agnhost:2.36",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									"memory": resource.MustParse("20Mi"),
								},
								Limits: v1.ResourceList{
									"memory": resource.MustParse("20Mi"),
								},
							},
							Command: []string{
								"/agnhost",
							},
							Args: []string{
								"serve-hostname",
								"--http",
								"--port",
								strconv.Itoa(AgnhostHTTPPort),
							},

							Ports: []v1.ContainerPort{
								{
									ContainerPort: AgnhostHTTPPort,
								},
							},
							Env: []v1.EnvVar{},
						},
					},
				},
			},
		},
	}
}
