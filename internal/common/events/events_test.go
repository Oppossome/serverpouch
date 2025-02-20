package events_test

import (
	"testing"
	"time"

	"oppossome/serverpouch/internal/common/events"

	"github.com/stretchr/testify/assert"
)

func channelGet[O any](t *testing.T, channel <-chan O) *O {
	select {
	case value, open := <-channel:
		if open {
			return &value
		}
	case <-time.After(time.Second * 5):
		assert.Fail(t, "Channel timeout exceeded.")
	}

	return nil
}

func TestEvents(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		testEvent := events.New[int]()
		chan1 := testEvent.On()
		chan2 := testEvent.On()

		go testEvent.Dispatch(1)
		assert.Equal(t, 1, *channelGet(t, chan1))
		assert.Equal(t, 1, *channelGet(t, chan2))

		// Off removes a specified channel.
		testEvent.Off(chan2)
		go testEvent.Dispatch(2)
		assert.Equal(t, 2, *channelGet(t, chan1))
		assert.Nil(t, channelGet(t, chan2))

		// Destroy closes all dependent channels.
		testEvent.Destroy()
		go testEvent.Dispatch(3)
		assert.Nil(t, channelGet(t, chan1))
		assert.Nil(t, channelGet(t, chan2))
	})
}
