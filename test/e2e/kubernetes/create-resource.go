package k8s

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

var (
	ErrUnknownResourceType = fmt.Errorf("unknown resource type")
	ErrNilResource         = fmt.Errorf("cannot create nil resource")
)

func CreateResource(ctx context.Context, obj runtime.Object, clientset *kubernetes.Clientset) error {
	// Create the object
	if obj == nil {
		return ErrNilResource
	}

	switch o := obj.(type) {
	case *appsv1.DaemonSet:
		log.Printf("Create/Update DaemonSet \"%s\" in namespace \"%s\"\n", o.Name, o.Namespace)
		client := clientset.AppsV1().DaemonSets(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return fmt.Errorf("failed to create DaemonSet \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return fmt.Errorf("failed to create/update DaemonSet \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)

	case *appsv1.Deployment:
		log.Printf("Create/Update Deployment \"%s\" in namespace \"%s\"\n", o.Name, o.Namespace)
		client := clientset.AppsV1().Deployments(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return fmt.Errorf("failed to create Deployment \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return fmt.Errorf("failed to create/update Deployment \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)

	case *v1.Service:
		log.Printf("Create/Update Service \"%s\" in namespace \"%s\"\n", o.Name, o.Namespace)
		client := clientset.CoreV1().Services(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return fmt.Errorf("failed to create Service \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return fmt.Errorf("failed to create/update Service \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)

	case *v1.ServiceAccount:
		log.Printf("Create/Update ServiceAccount \"%s\" in namespace \"%s\"\n", o.Name, o.Namespace)
		client := clientset.CoreV1().ServiceAccounts(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return fmt.Errorf("failed to create ServiceAccount \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return fmt.Errorf("failed to create/update ServiceAccount \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)

	case *rbacv1.Role:
		log.Printf("Create/Update Role \"%s\" in namespace \"%s\"\n", o.Name, o.Namespace)
		client := clientset.RbacV1().Roles(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return fmt.Errorf("failed to create Role \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return fmt.Errorf("failed to create/update Role \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)

	case *rbacv1.RoleBinding:
		log.Printf("Create/Update RoleBinding \"%s\" in namespace \"%s\"\n", o.Name, o.Namespace)
		client := clientset.RbacV1().RoleBindings(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return fmt.Errorf("failed to create RoleBinding \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return fmt.Errorf("failed to create/update RoleBinding \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)

	case *rbacv1.ClusterRole:
		log.Printf("Create/Update ClusterRole \"%s\"\n", o.Name)
		client := clientset.RbacV1().ClusterRoles()
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return fmt.Errorf("failed to create ClusterRole \"%s\": %w", o.Name, err)
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return fmt.Errorf("failed to create/update ClusterRole \"%s\": %w", o.Name, err)

	case *rbacv1.ClusterRoleBinding:
		log.Printf("Create/Update ClusterRoleBinding \"%s\"\n", o.Name)
		client := clientset.RbacV1().ClusterRoleBindings()
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return fmt.Errorf("failed to create ClusterRoleBinding \"%s\": %w", o.Name, err)
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return fmt.Errorf("failed to create/update ClusterRoleBinding \"%s\": %w", o.Name, err)

	case *v1.ConfigMap:
		log.Printf("Create/Update ConfigMap \"%s\" in namespace \"%s\"\n", o.Name, o.Namespace)
		client := clientset.CoreV1().ConfigMaps(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return fmt.Errorf("failed to create ConfigMap \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return fmt.Errorf("failed to create/update ConfigMap \"%s\" in namespace \"%s\": %w", o.Name, o.Namespace, err)

	default:
		return fmt.Errorf("unknown object type: %T, err: %w", obj, ErrUnknownResourceType)
	}
}
