package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// UpsertableResource defines a resource which can be upserted.
type UpsertableResource interface {
	GetName() string
	GetUID() types.UID
	SetUID(types.UID)
	GetResourceVersion() string
	SetResourceVersion(string)
}

// Upserter defines an interface which should be implemented by something which wants to create or update Kubernetes resources.
type Upserter[T UpsertableResource] interface {
	Create(ctx context.Context, obj T, opts metav1.CreateOptions) (T, error)
	Update(ctx context.Context, obj T, opts metav1.UpdateOptions) (T, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (T, error)
}

func UpsertResource(ctx context.Context, kubeClient kubernetes.Interface, resource metav1.Object) error {
	switch v := resource.(type) {
	case *corev1.ConfigMap:
		return upsert(ctx, kubeClient.CoreV1().ConfigMaps(resource.GetNamespace()), v)
	default:
		return fmt.Errorf("unsupported resource type: %T", resource)
	}
}

func upsert[U Upserter[T], T UpsertableResource](ctx context.Context, upserter U, obj T) error {
	_, err := upserter.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return fmt.Errorf("failed to get %T: %w", obj, err)
		}

		if _, err := upserter.Create(ctx, obj, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create %T: %w", obj, err)
		}
		return nil
	}

	obj, err = upserter.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update %T: %w", obj, err)
	}

	return nil
}
