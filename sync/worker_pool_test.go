package sync

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func Test_WorkerPool(t *testing.T) {
	t.Parallel()

	t.Run("golden", func(t *testing.T) {
		t.Parallel()
		const poolSize = 2
		wp := NewWorkerPool(t.Context(), "my-pool", poolSize, 50)

		tasksProcessed := atomic.NewInt32(0)
		done := [4]chan struct{}{
			make(chan struct{}),
			make(chan struct{}),
			make(chan struct{}),
			make(chan struct{}),
		}

		for j := range 4 {
			go func(jj int) {
				for range 100 {
					wp.SubmitBlocking(func(ctx context.Context) {
						// Use crypto/rand for secure random number generation
						n, err := rand.Int(rand.Reader, big.NewInt(100))
						if err != nil {
							t.Errorf("error occurred while submitting blocking: %v", err)
						}
						time.Sleep(time.Duration(n.Int64()) * time.Millisecond)
						tasksProcessed.Inc()
					})
				}

				close(done[jj])
			}(j)
		}

		<-done[0]
		<-done[1]
		<-done[2]
		<-done[3]

		wp.Close()
		require.Equal(t, int32(400), tasksProcessed.Load())
	})
}
