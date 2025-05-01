package k8s

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stest "k8s.io/client-go/testing"
)

func TestUpsertResource(t *testing.T) {
	t.Parallel()

	t.Run("configmap", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-configmap",
				Namespace: "default",
			},
		}

		err := UpsertResource(ctx, kubeClient, configMap)
		require.NoError(t, err)
	})

	t.Run("secret", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
		}

		err := UpsertResource(ctx, kubeClient, secret)
		require.NoError(t, err)
	})
}

func TestUpsertResource_UnsupportedResource(t *testing.T) {
	t.Parallel()

	kubeClient := fake.NewClientset()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	unsupportedResource := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	err := UpsertResource(ctx, kubeClient, unsupportedResource)
	require.Error(t, err)
	require.Equal(t, "unsupported resource type: *v1.Pod", err.Error())
}

func TestUpsert_Create(t *testing.T) {
	t.Parallel()

	t.Run("ConfigMap", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-configmap",
				Namespace: "default",
			},
		}

		kubeClient.PrependReactor("get", "configmaps", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, "test-configmap")
		})

		err := upsert(ctx, kubeClient.CoreV1().ConfigMaps("default"), configMap)
		require.NoError(t, err)
	})

	t.Run("Secret", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
		}

		kubeClient.PrependReactor("get", "secrets", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "secrets"}, "test-secret")
		})

		err := upsert(ctx, kubeClient.CoreV1().Secrets("default"), secret)
		require.NoError(t, err)
	})
}

func TestUpsert_Update(t *testing.T) {
	t.Parallel()

	t.Run("ConfigMap", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-configmap",
				Namespace: "default",
			},
		}

		_, _ = kubeClient.CoreV1().ConfigMaps("default").Create(ctx, configMap, metav1.CreateOptions{})

		err := upsert(ctx, kubeClient.CoreV1().ConfigMaps("default"), configMap)
		require.NoError(t, err)
	})

	t.Run("Secret", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
		}

		_, _ = kubeClient.CoreV1().Secrets("default").Create(ctx, secret, metav1.CreateOptions{})

		err := upsert(ctx, kubeClient.CoreV1().Secrets("default"), secret)
		require.NoError(t, err)
	})
}

func TestUpsert_GetError(t *testing.T) {
	t.Parallel()

	t.Run("ConfigMap", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-configmap",
				Namespace: "default",
			},
		}

		kubeClient.PrependReactor("get", "configmaps", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("get error")
		})

		err := upsert(ctx, kubeClient.CoreV1().ConfigMaps("default"), configMap)
		require.Error(t, err)
		require.Equal(t, "failed to get *v1.ConfigMap: get error", err.Error())
	})

	t.Run("Secret", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
		}

		kubeClient.PrependReactor("get", "secrets", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("get error")
		})

		err := upsert(ctx, kubeClient.CoreV1().Secrets("default"), secret)
		require.Error(t, err)
		require.Equal(t, "failed to get *v1.Secret: get error", err.Error())
	})
}

func TestUpsert_CreateError(t *testing.T) {
	t.Parallel()

	t.Run("ConfigMap", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-configmap",
				Namespace: "default",
			},
		}

		kubeClient.PrependReactor("get", "configmaps", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, "test-configmap")
		})

		kubeClient.PrependReactor("create", "configmaps", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("create error")
		})

		err := upsert(ctx, kubeClient.CoreV1().ConfigMaps("default"), configMap)
		require.Error(t, err)
		require.Equal(t, "failed to create *v1.ConfigMap: create error", err.Error())
	})

	t.Run("Secret", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
		}

		kubeClient.PrependReactor("get", "secrets", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "secrets"}, "test-secret")
		})

		kubeClient.PrependReactor("create", "secrets", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("create error")
		})

		err := upsert(ctx, kubeClient.CoreV1().Secrets("default"), secret)
		require.Error(t, err)
		require.Equal(t, "failed to create *v1.Secret: create error", err.Error())
	})
}

func TestUpsert_UpdateError(t *testing.T) {
	t.Parallel()

	t.Run("ConfigMap", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-configmap",
				Namespace: "default",
			},
		}

		_, _ = kubeClient.CoreV1().ConfigMaps("default").Create(ctx, configMap, metav1.CreateOptions{})

		kubeClient.PrependReactor("update", "configmaps", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("update error")
		})

		err := upsert(ctx, kubeClient.CoreV1().ConfigMaps("default"), configMap)
		require.Error(t, err)
		require.Equal(t, "failed to update *v1.ConfigMap: update error", err.Error())
	})

	t.Run("Secret", func(t *testing.T) {
		t.Parallel()

		kubeClient := fake.NewClientset()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
		}

		_, _ = kubeClient.CoreV1().Secrets("default").Create(ctx, secret, metav1.CreateOptions{})
		kubeClient.PrependReactor("update", "secrets", func(action k8stest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("update error")
		})
		err := upsert(ctx, kubeClient.CoreV1().Secrets("default"), secret)
		require.EqualError(t, err, "failed to update *v1.Secret: update error")
	})
}
