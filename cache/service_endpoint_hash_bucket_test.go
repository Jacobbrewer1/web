package cache

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jacobbrewer1/web/slices"
	"github.com/serialx/hashring"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_Lifecycle(t *testing.T) {
	t.Parallel()

	ep := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-svc",
			Namespace: "my-ns",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						TargetRef: &corev1.ObjectReference{
							Name: "a",
						},
					},
					{
						TargetRef: &corev1.ObjectReference{
							Name: "b",
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := slog.New(slog.DiscardHandler)
	k := fake.NewClientset(ep)
	sb := NewServiceEndpointHashBucket(l, k, "my-svc", "my-ns", "")
	require.NoError(t, sb.Start(ctx))

	// Check that we can receive an endpoint update after starting.
	ep.Subsets = append(ep.Subsets, corev1.EndpointSubset{
		Addresses: []corev1.EndpointAddress{
			{
				TargetRef: &corev1.ObjectReference{
					Name: "c",
				},
			},
		},
	})
	_, err := k.CoreV1().Endpoints("my-ns").Update(ctx, ep, metav1.UpdateOptions{})
	require.NoError(t, err)

	require.Eventually(
		t,
		func() bool {
			nodes, ok := sb.hr.GetNodes("", sb.hr.Size())
			return ok && len(nodes) == 3
		},
		5*time.Second,
		10*time.Millisecond,
	)

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
		l: slog.New(slog.DiscardHandler),
		hr: hashring.New([]string{
			"a",
			"b",
		}),
	}

	sb.onEndpointUpdate(
		&corev1.Endpoints{
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							TargetRef: &corev1.ObjectReference{
								Name: "a",
							},
						},
						{
							TargetRef: &corev1.ObjectReference{
								Name: "b",
							},
						},
					},
				},
			},
		},
		&corev1.Endpoints{
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							TargetRef: &corev1.ObjectReference{
								Name: "b",
							},
						},
					},
				},
				{
					Addresses: []corev1.EndpointAddress{
						{
							TargetRef: &corev1.ObjectReference{
								Name: "c",
							},
						},
					},
				},
			},
		},
	)

	nodes, ok := sb.hr.GetNodes("", sb.hr.Size())
	require.True(t, ok)
	require.Equal(
		t,
		[]string{
			"b",
			"c",
		},
		nodes,
	)
}

func Test_endpointsToSet(t *testing.T) {
	t.Parallel()
	require.Equal(
		t,
		slices.NewSet("a", "b", "c"),
		endpointsToSet(
			&corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								TargetRef: &corev1.ObjectReference{
									Name: "a",
								},
							},
						},
					},
					{
						Addresses: []corev1.EndpointAddress{
							{
								TargetRef: &corev1.ObjectReference{
									Name: "b",
								},
							},
							{
								TargetRef: &corev1.ObjectReference{
									Name: "c",
								},
							},
						},
					},
				},
			},
		),
	)
}
