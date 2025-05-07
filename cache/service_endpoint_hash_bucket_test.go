package cache

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/serialx/hashring"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/jacobbrewer1/web/k8s"
	"github.com/jacobbrewer1/web/slices"
)

func Test_Lifecycle(t *testing.T) {
	t.Parallel()

	// Create an EndpointSlice instead of Endpoints
	endpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:         "my-app-name-n4wmx",
			GenerateName: "my-app-name-",
			Namespace:    k8s.DeployedNamespace(),
			Labels: map[string]string{
				"app.kubernetes.io/component":            "application",
				"app.kubernetes.io/instance":             "my-app-name",
				"app.kubernetes.io/managed-by":           "Helm",
				"app.kubernetes.io/name":                 "my-app-name",
				"app.kubernetes.io/version":              "05ed3af",
				"endpointslice.kubernetes.io/managed-by": "endpointslice-controller.k8s.io",
				"kubernetes.io/service-name":             "my-app-name",
			},
			Annotations: map[string]string{
				"endpoints.kubernetes.io/last-change-trigger-time": time.Now().Format(time.RFC3339),
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				TargetRef: &corev1.ObjectReference{
					Kind: "Pod",
					Name: "a",
				},
			},
			{
				TargetRef: &corev1.ObjectReference{
					Kind: "Pod",
					Name: "b",
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := slog.New(slog.DiscardHandler)
	k := fake.NewClientset(endpointSlice)
	sb := NewServiceEndpointHashBucket(l, k, "my-app-name", k8s.DeployedNamespace(), k8s.PodName())
	require.NoError(t, sb.Start(ctx))

	// Check that we can receive an endpoint update after starting.
	endpointSlice.Endpoints = append(endpointSlice.Endpoints, discoveryv1.Endpoint{
		TargetRef: &corev1.ObjectReference{
			Kind: "Pod",
			Name: "c",
		},
	})

	_, err := k.DiscoveryV1().EndpointSlices(k8s.DeployedNamespace()).Update(ctx, endpointSlice, metav1.UpdateOptions{})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		nodes, ok := sb.hr.GetNodes("", sb.hr.Size())
		return ok && len(nodes) == 3
	}, 5*time.Second, 10*time.Millisecond)

	// Check that a key lands in one of three buckets.
	hits := 0

	sb.thisPod = "a"
	if sb.InBucket("key") {
		hits++
	}

	sb.thisPod = "b"
	if sb.InBucket("key") {
		hits++
	}

	sb.thisPod = "c"
	if sb.InBucket("key") {
		hits++
	}

	require.Equal(t, 1, hits)
}

func Test_onEndpointUpdate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		sb := &ServiceEndpointHashBucket{
			mut: new(sync.RWMutex),
			l:   slog.New(slog.DiscardHandler),
			hr: hashring.New([]string{
				"a",
				"b",
			}),
			appName:      "my-app-name",
			appNamespace: k8s.DeployedNamespace(),
			thisPod:      k8s.PodName(),
		}

		sb.onEndpointUpdate(
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "my-app-name-n4wmx",
					Namespace:    k8s.DeployedNamespace(),
					GenerateName: "my-app-name-",
					Labels: map[string]string{
						"app.kubernetes.io/name": "my-app-name",
					},
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "a",
						},
					},
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "b",
						},
					},
				},
			},
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "my-app-name-n4wmx",
					Namespace:    k8s.DeployedNamespace(),
					GenerateName: "my-app-name-",
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "b",
						},
					},
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "c",
						},
					},
				},
			},
		)

		nodes, ok := sb.hr.GetNodes("", sb.hr.Size())
		require.True(t, ok)
		require.Equal(t, []string{
			"b",
			"c",
		}, nodes)
	})

	t.Run("success-no-generate-name", func(t *testing.T) {
		t.Parallel()

		sb := &ServiceEndpointHashBucket{
			mut: new(sync.RWMutex),
			l:   slog.New(slog.DiscardHandler),
			hr: hashring.New([]string{
				"a",
				"b",
			}),
			appName:      "my-app-name",
			appNamespace: k8s.DeployedNamespace(),
			thisPod:      k8s.PodName(),
		}

		sb.onEndpointUpdate(
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app-name-n4wmx",
					Namespace: k8s.DeployedNamespace(),
					Labels: map[string]string{
						"app.kubernetes.io/name": "my-app-name",
					},
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "a",
						},
					},
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "b",
						},
					},
				},
			},
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app-name-n4wmx",
					Namespace: k8s.DeployedNamespace(),
					Labels: map[string]string{
						"app.kubernetes.io/name": "my-app-name",
					},
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "b",
						},
					},
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "c",
						},
					},
				},
			},
		)

		nodes, ok := sb.hr.GetNodes("", sb.hr.Size())
		require.True(t, ok)
		require.Equal(t, []string{
			"b",
			"c",
		}, nodes)
	})

	t.Run("different-namespace", func(t *testing.T) {
		t.Parallel()

		sb := &ServiceEndpointHashBucket{
			mut: new(sync.RWMutex),
			l:   slog.New(slog.DiscardHandler),
			hr: hashring.New([]string{
				"a",
				"b",
			}),
			appName:      "my-app-name",
			appNamespace: k8s.DeployedNamespace(),
			thisPod:      k8s.PodName(),
		}

		sb.onEndpointUpdate(
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "my-app-name-n4wmx",
					Namespace:    "other-namespace",
					GenerateName: "my-app-name-",
					Labels: map[string]string{
						"app.kubernetes.io/name": "my-app-name",
					},
				},
				Endpoints: []discoveryv1.Endpoint{},
			},
			nil,
		)

		nodes, ok := sb.hr.GetNodes("", sb.hr.Size())
		require.True(t, ok)
		require.Equal(t, []string{
			"b",
			"a",
		}, nodes)
	})

	t.Run("different-app", func(t *testing.T) {
		t.Parallel()

		sb := &ServiceEndpointHashBucket{
			mut: new(sync.RWMutex),
			l:   slog.New(slog.DiscardHandler),
			hr: hashring.New([]string{
				"a",
				"b",
			}),
			appName:      "my-app-name",
			appNamespace: k8s.DeployedNamespace(),
			thisPod:      k8s.PodName(),
		}

		sb.onEndpointUpdate(
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "other-app-name-n4wmx",
					Namespace:    k8s.DeployedNamespace(),
					GenerateName: "other-app-name-",
					Labels: map[string]string{
						"app.kubernetes.io/name": "other-app-name",
					},
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "a",
						},
					},
				},
			},
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "other-app-name-n4wmx",
					Namespace:    k8s.DeployedNamespace(),
					GenerateName: "other-app-name-",
					Labels: map[string]string{
						"app.kubernetes.io/name": "other-app-name",
					},
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "c",
						},
					},
				},
			},
		)

		nodes, ok := sb.hr.GetNodes("", sb.hr.Size())
		require.True(t, ok)
		require.Equal(t, []string{
			"b",
			"a",
		}, nodes)
	})
}

func Test_endpointSliceToSet(t *testing.T) {
	t.Parallel()
	require.Equal(
		t,
		slices.NewSet("a", "b", "c"),
		endpointSliceToSet(
			&discoveryv1.EndpointSlice{
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "a",
						},
					},
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "b",
						},
					},
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "c",
						},
					},
				},
			},
		),
	)
}

func Benchmark_OnEndpointUpdate(b *testing.B) {
	sb := &ServiceEndpointHashBucket{
		mut:          new(sync.RWMutex),
		l:            slog.New(slog.DiscardHandler),
		hr:           hashring.New([]string{}),
		appName:      "my-app-name",
		appNamespace: k8s.DeployedNamespace(),
		thisPod:      k8s.PodName(),
	}

	newUUID := func() string {
		uid := uuid.New().String()
		sb.hr.AddNode(uid)
		return uid
	}

	b.ResetTimer()

	for range b.N {
		sb.onEndpointUpdate(
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "my-app-name-n4wmx",
					Namespace:    k8s.DeployedNamespace(),
					GenerateName: "my-app-name-",
					Labels: map[string]string{
						"app.kubernetes.io/name": "my-app-name",
					},
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: newUUID(),
						},
					},
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: newUUID(),
						},
					},
				},
			},
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "my-app-name-n4wmx",
					Namespace:    k8s.DeployedNamespace(),
					GenerateName: "my-app-name-",
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: uuid.New().String(),
						},
					},
					{
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: uuid.New().String(),
						},
					},
				},
			},
		)
	}
}

func Test_InBucket_WithoutStart(t *testing.T) {
	t.Parallel()

	sb := NewServiceEndpointHashBucket(
		slog.New(slog.DiscardHandler),
		fake.NewClientset(),
		"my-app-name",
		k8s.DeployedNamespace(),
		k8s.PodName(),
	)

	require.False(t, sb.InBucket("key"), "should not be in bucket before starting")
}
