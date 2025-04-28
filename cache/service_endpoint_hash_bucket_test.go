package cache

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

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
			Name:      "my-svc",
			Namespace: "my-ns",
			Labels: map[string]string{
				"kubernetes.io/service-name": "my-svc",
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
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := slog.New(slog.DiscardHandler)
	k := fake.NewClientset(endpointSlice)
	sb := NewServiceEndpointHashBucket(l, k, "my-svc", "my-ns", "")
	require.NoError(t, sb.Start(ctx))

	// Check that we can receive an endpoint update after starting.
	endpointSlice.Endpoints = append(endpointSlice.Endpoints, discoveryv1.Endpoint{
		TargetRef: &corev1.ObjectReference{
			Kind: "Pod",
			Name: "c",
		},
	})

	_, err := k.DiscoveryV1().EndpointSlices("my-ns").Update(ctx, endpointSlice, metav1.UpdateOptions{})
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

	sb := &ServiceEndpointHashBucket{
		mut: new(sync.RWMutex),
		l:   slog.New(slog.DiscardHandler),
		hr: hashring.New([]string{
			"a",
			"b",
		}),
		appName:      "my-svc",
		appNamespace: "my-ns",
	}

	sb.onEndpointUpdate(
		&discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-svc",
				Namespace: "my-ns",
				Labels: map[string]string{
					"kubernetes.io/service-name": "my-svc",
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
				Name:      "my-svc",
				Namespace: "my-ns",
				Labels: map[string]string{
					"kubernetes.io/service-name": "my-svc",
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

func Test_InBucket_WithoutStart(t *testing.T) {
	t.Parallel()

	sb := NewServiceEndpointHashBucket(
		slog.New(slog.DiscardHandler),
		fake.NewClientset(),
		"my-svc",
		"my-ns",
		k8s.PodName(),
	)

	require.False(t, sb.InBucket("key"), "should not be in bucket before starting")
}
