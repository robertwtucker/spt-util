//
// Copyright (c) 2023 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package eventbus_test

import (
	"sync"
	"testing"

	"github.com/robertwtucker/spt-util/pkg/eventbus"
	"github.com/stretchr/testify/assert"
)

func TestEventBus_NewEventBus(t *testing.T) {
	if eb := eventbus.NewEventBus(); eb == nil {
		t.Fail()
	}
}

func TestEventBus_NewEventChannel(t *testing.T) {
	if ec := eventbus.NewEventChannel(); ec == nil {
		t.Fail()
	}
}

func TestEventBus_SubscribeEvent(t *testing.T) {
	eventName := "foo"

	eb := eventbus.NewEventBus()
	_ = eb.SubscribeEvent(eventName)

	assert.Equal(t, true, eb.HasSubscribers(eventName))
}

func TestEventBus_SubscribeEventChannel(t *testing.T) {
	eventName := "foo"

	eb := eventbus.NewEventBus()
	ec := eventbus.NewEventChannel()
	eb.SubscribeEventChannel(ec, eventName)

	assert.Equal(t, true, eb.HasSubscribers(eventName))
}

func TestEventBus_PublishEvent(t *testing.T) {
	eventName := "foo"
	eventData := "bar"

	eb := eventbus.NewEventBus()
	ec := eb.SubscribeEvent(eventName)

	go func() {
		event := <-ec
		defer event.Done()

		assert.Equal(t, eventName, event.Name)
		assert.Equal(t, eventData, event.Data)
	}()

	eb.PublishEvent(eventName, eventData)
}

func TestEventBus_PublishEventAsync(t *testing.T) {
	eventName := "foo"
	eventData := "bar"

	eb := eventbus.NewEventBus()
	ec := eb.SubscribeEvent(eventName)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		event := <-ec
		assert.Equal(t, eventName, event.Name)
		assert.Equal(t, eventData, event.Data)
		wg.Done()
	}()

	eb.PublishEventAsync(eventName, eventData)
	wg.Wait()
}

func TestEventBus_SubscribeEventCallback(t *testing.T) {
	eventName := "foo"
	eventData := "bar"

	eb := eventbus.NewEventBus()
	eb.SubscribeEventCallback(
		eventName,
		func(name string, data interface{}) {
			assert.Equal(t, eventName, name)
			assert.Equal(t, eventData, data)
		})

	eb.PublishEvent(eventName, eventData)
}
