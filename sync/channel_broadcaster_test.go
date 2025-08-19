package sync

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChannelBroadcaster(t *testing.T) {
	t.Parallel()

	t.Run("lifecycle golden close", func(t *testing.T) {
		t.Parallel()

		logger := slog.New(slog.DiscardHandler)
		broadcaster := NewChannelBroadcaster[string](logger)

		// Start the broadcaster
		broadcaster.Start(t.Context())

		// Register a channel
		ch := make(chan string, 1)
		broadcaster.AddChannel("test-channel", ch)

		broadcaster.GetBroadcastingChannel() <- "Hello, World!"

		// Check if the message was received, using Eventually to allow time for goroutine to spin up
		require.Eventually(t, func() bool {
			select {
			case msg := <-ch:
				return msg == "Hello, World!"
			default:
				return false
			}
		}, 1*time.Second, 100*time.Millisecond, "expected message to be received by the channel")

		// Stop the broadcaster
		broadcaster.Close()
	})

	t.Run("removing channel", func(t *testing.T) {
		t.Parallel()

		var ch1 = make(chan string, 1)
		var ch2 = make(chan string, 1)

		logger := slog.New(slog.DiscardHandler)
		broadcaster := NewChannelBroadcaster[string](logger)

		broadcaster.Start(t.Context())

		broadcaster.AddChannel("channel1", ch1)
		broadcaster.AddChannel("channel2", ch2)

		broadcaster.GetBroadcastingChannel() <- "Test Message"
		require.Eventually(t, func() bool {
			select {
			case msg := <-ch1:
				return msg == "Test Message"
			default:
				return false
			}
		}, 1*time.Second, 100*time.Millisecond, "expected message to be received by channel1")
		require.Eventually(t, func() bool {
			select {
			case msg := <-ch2:
				return msg == "Test Message"
			default:
				return false
			}
		}, 1*time.Second, 100*time.Millisecond, "expected message to be received by channel2")

		// Remove channel1
		broadcaster.RemoveChannel("channel1")

		// Broadcast another message
		broadcaster.GetBroadcastingChannel() <- "Test Message Again"

		require.Never(t, func() bool {
			select {
			case msg := <-ch1:
				return msg == "Test Message Again"
			default:
				return false
			}
		}, 1*time.Second, 100*time.Millisecond, "expected channel1 to not receive any more messages after removal")
		require.Eventually(t, func() bool {
			select {
			case msg := <-ch2:
				return msg == "Test Message Again"
			default:
				return false
			}
		}, 1*time.Second, 100*time.Millisecond, "expected channel2 to still receive messages after channel1 removal")
	})

	t.Run("broadcasting channel full", func(t *testing.T) {
		t.Parallel()

		logger := slog.New(slog.DiscardHandler)
		broadcaster := NewChannelBroadcaster[string](logger)

		// Start the broadcaster
		broadcaster.Start(t.Context())

		// Register a channel with a small buffer
		ch := make(chan string, 1)
		broadcaster.AddChannel("test-channel", ch)

		// Send a message to the broadcasting channel
		broadcaster.GetBroadcastingChannel() <- "Hello, World!"

		// Try to send another message to the broadcasting channel
		broadcaster.GetBroadcastingChannel() <- "Another Message"

		require.Never(t, func() bool {
			select {
			case msg := <-ch:
				return msg == "Another Message"
			default:
				return false
			}
		}, 1*time.Second, 100*time.Millisecond, "expected channel to not receive any more messages after it is full")
	})
}
