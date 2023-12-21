package kubernetes

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/Azure/azure-container-networking/test/integration/networkobservability/types"
	"github.com/Azure/azure-container-networking/test/integration/networkobservability/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type CreateKapingerDeployment struct {
	KapingerNamespace string
	KapingerReplicas  string
}

func (c *CreateKapingerDeployment) Run(values *types.JobValues) error {
	// Path to the kubeconfig file, leave empty for in-cluster config
	kubeconfigPath := "" // Set your kubeconfig path if needed

	// Create a Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Println("Error building kubeconfig: ", err)
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("Error creating Kubernetes client: ", err)
		return err
	}

	// Create a sample Deployment object
	deployment := c.getKapingerDeployment()
	_, err = clientset.AppsV1().Deployments("default").Create(context.TODO(), deployment, metaV1.CreateOptions{})
	if err != nil {
		fmt.Println("Error creating Deployment: ", err)
		return err
	}

	// Print the YAML to stdout
	return nil
}

func (c *CreateKapingerDeployment) Prevalidate(values *types.JobValues) error {

	_, err := strconv.Atoi(c.KapingerReplicas)
	if err != nil {
		fmt.Println("Error converting replicas to int for Kapinger replicas: ", err)
		return err
	}

	return nil
}

func (c *CreateKapingerDeployment) DryRun(values *types.JobValues) error {
	return nil
}

func GenerateKapingerYAML(folder string) {
	kappiefolder := folder + "/kapinger"
	os.MkdirAll(kappiefolder, os.ModePerm)
	// Create a sample Deployment object
	c := CreateKapingerDeployment{
		KapingerNamespace: "default",
		KapingerReplicas:  "1",
	}

	resources := map[string]interface{}{
		"kapinger-deployment.yaml":         c.getKapingerDeployment(),
		"kapinger-service.yaml":            c.getKapingerService(),
		"kapinger-serviceaccount.yaml":     c.getKapingerServiceAccount(),
		"kapinger-clusterrole.yaml":        c.getKapingerClusterRole(),
		"kapinger-clusterrolebinding.yaml": c.getKapingerClusterRoleBinding(),
	}

	for filename, obj := range resources {
		yamlBytes, err := yaml.Marshal(obj)
		if err != nil {
			fmt.Println("Error marshalling object: ", err)
		}
		err = utils.WriteYAMLToFile(yamlBytes, kappiefolder+"/"+filename)
		if err != nil {
			fmt.Println("Error writing YAML to file: ", err)
		}
	}
}

func (c *CreateKapingerDeployment) getKapingerDeployment() *appsv1.Deployment {
	replicas, err := strconv.Atoi(c.KapingerReplicas)
	if err != nil {
		fmt.Println("Error converting replicas to int for Kapinger replicas: ", err)
	}
	reps := int32(replicas)
	return &appsv1.Deployment{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "kapinger",
			Namespace: c.KapingerNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &reps,
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kapinger",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"app":    "kapinger",
						"server": "good",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "kapinger-sa",
					Containers: []v1.Container{
						{
							Name:  "kapinger",
							Image: "acnpublic.azurecr.io/kapinger:be57650",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									"memory": resource.MustParse("20Mi"),
								},
								Limits: v1.ResourceList{
									"memory": resource.MustParse("20Mi"),
								},
							},
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
							Env: []v1.EnvVar{
								{
									Name:  "TARGET_TYPE",
									Value: "service",
								},
								{
									Name:  "HTTP_PORT",
									Value: "8080",
								},
								{
									Name:  "TCP_PORT",
									Value: "8085",
								},
								{
									Name:  "UDP_PORT",
									Value: "8086",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *CreateKapingerDeployment) getKapingerService() *v1.Service {
	return &v1.Service{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "kapinger-service",
			Namespace: c.KapingerNamespace,
			Labels: map[string]string{
				"app": "kapinger",
			},
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "kapinger",
			},
			Ports: []v1.ServicePort{
				{
					Port:       8080,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
}

func (c *CreateKapingerDeployment) getKapingerServiceAccount() *v1.ServiceAccount {
	return &v1.ServiceAccount{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "kapinger-sa",
			Namespace: c.KapingerNamespace,
		},
	}
}

func (c *CreateKapingerDeployment) getKapingerClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "kapinger-role",
			Namespace: c.KapingerNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services", "pods"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
}

func (c *CreateKapingerDeployment) getKapingerClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "kapinger-rolebinding",
			Namespace: c.KapingerNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kapinger-sa",
				Namespace: c.KapingerNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kapinger-role",
		},
	}
}
