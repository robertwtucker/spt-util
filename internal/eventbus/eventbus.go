//
// Copyright (c) 2023 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package eventbus

import (
	"sync"
)

// Event holds the name of an event and its associated data.
type Event struct {
	Data interface{}
	Name string
	wg   *sync.WaitGroup
}

// Done wraps the WaitGroup.Done() call.
func (e *Event) Done() {
	if e.wg != nil {
		e.wg.Done()
	}
}

// EventChannel is a channel that accepts an Event.
type EventChannel chan Event

// NewEventChannel creates a new, unbuffered EventChannel.
func NewEventChannel() EventChannel {
	return make(EventChannel)
}

// eventChannelSlice is a slice of channels that accept an Event.
type eventChannelSlice []EventChannel

// CallbackFunction defines a callback function for the named Event.
type CallbackFunction func(name string, data interface{})

// EventBus stores the mapping of subscribers (instances of EventChannel)
// to a corresponding event name.
type EventBus struct {
	mutex       sync.RWMutex
	subscribers map[string]eventChannelSlice
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string]eventChannelSlice),
	}
}

// getEventSubscribers returns the EventChannel(s) subscribed the named Event.
func (eb *EventBus) getEventSubscribers(name string) eventChannelSlice {
	subscribers := eventChannelSlice{}

	if len(eb.subscribers[name]) > 0 {
		subscribers = append(subscribers, eb.subscribers[name]...)
	}

	return subscribers
}

// PublishEvent sends data to all named Event subscribers. It waits
// for all subscribers to finish (each must call Done() on Event).
func (eb *EventBus) PublishEvent(name string, data interface{}) {
	wg := sync.WaitGroup{}
	subscribers := eb.getEventSubscribers(name)
	wg.Add(len(subscribers))

	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	go func(eventChannels eventChannelSlice, event Event) {
		for _, ec := range eventChannels {
			ec <- event
		}
		wg.Done()
	}(subscribers, Event{Data: data, Name: name, wg: &wg})

	wg.Wait()
}

// SubscribeEvent returns an EventChannel subscribed to the named Event.
func (eb *EventBus) SubscribeEvent(name string) EventChannel {
	ec := NewEventChannel()
	eb.SubscribeEventChannel(ec, name)

	return ec
}

// SubscribeEventCallback registers a callback in response to an Event.
func (eb *EventBus) SubscribeEventCallback(name string, callback CallbackFunction) {
	ec := NewEventChannel()
	eb.SubscribeEventChannel(ec, name)

	go func(callback CallbackFunction) {
		event := <-ec
		callback(event.Name, event.Data)
		event.Done()
	}(callback)
}

// SubscribeEventChannel registers an EventChannel's subscription to an Event.
func (eb *EventBus) SubscribeEventChannel(ec EventChannel, name string) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if subscribers, found := eb.subscribers[name]; found {
		eb.subscribers[name] = append(subscribers, ec)
	} else {
		eb.subscribers[name] = append(eventChannelSlice{}, ec)
	}
}
