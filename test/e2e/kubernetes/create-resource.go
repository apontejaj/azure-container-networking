package kubernetes

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

func CreateResource(ctx context.Context, obj runtime.Object, clientset *kubernetes.Clientset) error {
	// Create the object
	switch o := obj.(type) {
	case *appsv1.DaemonSet:
		client := clientset.AppsV1().DaemonSets(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return err
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return err

	case *appsv1.Deployment:
		client := clientset.AppsV1().Deployments(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return err
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return err

	case *v1.ServiceAccount:
		client := clientset.CoreV1().ServiceAccounts(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return err
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return err

	case *rbacv1.Role:
		client := clientset.RbacV1().Roles(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return err
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return err

	case *rbacv1.RoleBinding:
		client := clientset.RbacV1().RoleBindings(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return err
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return err

	case *rbacv1.ClusterRole:
		client := clientset.RbacV1().ClusterRoles()
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return err
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return err

	case *rbacv1.ClusterRoleBinding:
		client := clientset.RbacV1().ClusterRoleBindings()
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return err
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return err

	case *v1.ConfigMap:
		client := clientset.CoreV1().ConfigMaps(o.Namespace)
		_, err := client.Get(ctx, o.Name, metaV1.GetOptions{})
			if errors.IsNotFound(err) {
			_, err = client.Create(ctx, o, metaV1.CreateOptions{})
			return err
		}
		_, err = client.Update(ctx, o, metaV1.UpdateOptions{})
		return err

	default:
		fmt.Println("The object is not a ServiceAccount, Role, or ClusterRole")
	}
	return nil
}
