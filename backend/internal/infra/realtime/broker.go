package realtime

import (
	"sync"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

const subscriberBuffer = 32

type subscriber struct {
	events chan model.Event
}

type Broker struct {
	mutex sync.Mutex
	rooms map[string]map[*subscriber]struct{}
}

func NewBroker() *Broker {
	return &Broker{rooms: map[string]map[*subscriber]struct{}{}}
}

func (broker *Broker) Subscribe(code string) (<-chan model.Event, func()) {
	member := &subscriber{events: make(chan model.Event, subscriberBuffer)}

	broker.mutex.Lock()
	defer broker.mutex.Unlock()

	members, found := broker.rooms[code]
	if !found {
		members = map[*subscriber]struct{}{}
		broker.rooms[code] = members
	}

	members[member] = struct{}{}

	unsubscribe := func() {
		broker.mutex.Lock()
		defer broker.mutex.Unlock()

		broker.remove(code, member)
	}

	return member.events, unsubscribe
}

func (broker *Broker) Publish(code string, event model.Event) {
	broker.mutex.Lock()
	defer broker.mutex.Unlock()

	for member := range broker.rooms[code] {
		select {
		case member.events <- event:
		default:
			broker.remove(code, member)
		}
	}
}

func (broker *Broker) CloseRoom(code string, reason model.CloseReason) {
	event := model.RoomClosedEvent(reason)

	broker.mutex.Lock()
	defer broker.mutex.Unlock()

	for member := range broker.rooms[code] {
		select {
		case member.events <- event:
		default:
		}

		broker.remove(code, member)
	}
}

func (broker *Broker) remove(code string, member *subscriber) {
	members, found := broker.rooms[code]
	if !found {
		return
	}

	_, subscribed := members[member]
	if !subscribed {
		return
	}

	delete(members, member)
	close(member.events)

	if len(members) == 0 {
		delete(broker.rooms, code)
	}
}
