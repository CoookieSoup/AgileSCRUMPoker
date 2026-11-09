package realtime_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/infra/realtime"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

func drain(events <-chan model.Event) []model.Event {
	received := []model.Event{}
	for event := range events {
		received = append(received, event)
	}

	return received
}

func TestPublishFansOutWithinRoomOnly(t *testing.T) {
	broker := realtime.NewBroker()
	first, unsubscribeFirst := broker.Subscribe("ABC234")
	second, unsubscribeSecond := broker.Subscribe("ABC234")
	other, unsubscribeOther := broker.Subscribe("XYZ789")
	event := model.ParticipantVotedEvent("guest")

	broker.Publish("ABC234", event)
	unsubscribeFirst()
	unsubscribeSecond()
	unsubscribeOther()

	assert.Equal(t, []model.Event{event}, drain(first))
	assert.Equal(t, []model.Event{event}, drain(second))
	assert.Empty(t, drain(other))
}

func TestCloseRoomSendsReasonAndEndsStream(t *testing.T) {
	broker := realtime.NewBroker()
	events, unsubscribe := broker.Subscribe("ABC234")

	broker.CloseRoom("ABC234", model.CloseReasonExpired)

	assert.Equal(t, []model.Event{model.RoomClosedEvent(model.CloseReasonExpired)}, drain(events))
	assert.NotPanics(t, unsubscribe)
}

func TestSlowSubscriberIsDropped(t *testing.T) {
	broker := realtime.NewBroker()
	slow, unsubscribeSlow := broker.Subscribe("ABC234")
	fast, unsubscribeFast := broker.Subscribe("ABC234")
	fastCount := 0

	for range 100 {
		broker.Publish("ABC234", model.ParticipantVotedEvent("guest"))

		select {
		case <-fast:
			fastCount++
		default:
		}
	}

	unsubscribeFast()
	unsubscribeSlow()

	assert.Equal(t, 100, fastCount)
	assert.Len(t, drain(slow), 32)
}

func TestUnsubscribeIsIdempotent(t *testing.T) {
	broker := realtime.NewBroker()
	events, unsubscribe := broker.Subscribe("ABC234")

	unsubscribe()
	unsubscribe()
	broker.Publish("ABC234", model.ParticipantVotedEvent("guest"))

	_, open := <-events

	assert.False(t, open)
}

func TestConcurrentUse(t *testing.T) {
	broker := realtime.NewBroker()

	var group sync.WaitGroup

	for range 20 {
		group.Go(func() {
			events, unsubscribe := broker.Subscribe("ABC234")
			defer unsubscribe()

			broker.Publish("ABC234", model.ParticipantVotedEvent("guest"))

			select {
			case <-events:
			default:
			}
		})
	}

	group.Wait()
	broker.CloseRoom("ABC234", model.CloseReasonClosed)

	events, unsubscribe := broker.Subscribe("ABC234")
	defer unsubscribe()

	require.NotNil(t, events)
}
