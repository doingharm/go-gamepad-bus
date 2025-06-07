package gamepads

import (
	"context"
	"errors"
	"runtime"
	"sync"
)

// bus is the internal implementation of the Bus interface.
type bus struct {
	sync.RWMutex
	eventChannel chan *Event
	errChannel   chan error
	notifier     notify
	channels     []*EventChannel
}

// Bus is the main interface for interacting with the joystick events.
type Bus interface {
	// NewEventChannel creates a new event channel that filters the events based on the provided filter functions.
	NewEventChannel(filters ...FilterFunc) (dest *EventChannel)
	// Gamepads returns a list of all the available gamepads connected to the system.
	Gamepads() (gamepads []Gamepad)
	// Subscribe subscribes to the joystick events for a specific gamepad ID.
	Subscribe(id string) (err error)
	// Unsubscribe unsubscribes from the joystick events for a specific gamepad ID.
	Unsubscribe(id string) (err error)
	// Close stops the bus and closes all the event channels.
	Close()
}

// This function creates a new gamepad bus instance and returns it, along with an error channel for any errors that may occur during the initialization process.
func New() (b Bus, errCh <-chan error, err error) {

	dest := &bus{
		eventChannel: make(chan *Event),
		errChannel:   make(chan error),
	}

	switch runtime.GOOS {
	case "linux":
		if dest.notifier, err = linuxNotifier(dest.eventChannel, dest.errChannel); err != nil {
			return
		}
		return dest, dest.errChannel, nil
	default:
		return nil, nil, errors.New(ErrOsNotSupported)
	}

}

func (b *bus) NewEventChannel(filters ...FilterFunc) (dest *EventChannel) {

	if b.notifier == nil {
		return nil
	}

	wg := sync.WaitGroup{}

	ctx, cancelFunc := context.WithCancel(context.Background())

	dest = &EventChannel{
		Ctx:        context.Background(),
		Ch:         make(chan *Event),
		CancelFunc: cancelFunc,
	}

	b.Lock()
	b.channels = append(b.channels, dest)
	b.Unlock()

	wg.Add(1)
	go func() {
		wg.Done()
		for {
			select {
			case <-ctx.Done():
				close(dest.Ch)
				dest.Ch = nil
				b.Lock()
				var clean []*EventChannel
				for _, channel := range b.channels {
					if channel.Ch != nil {
						clean = append(clean, channel)
					}
				}
				b.channels = clean
				b.Unlock()
				return
			case event := <-b.eventChannel:
				skip := false
				for _, filter := range filters {
					if !filter(event) {
						skip = true
					}
				}
				if skip {
					continue
				}
				dest.Ch <- event
			}
		}
	}()

	wg.Wait()
	return
}

func (b *bus) Gamepads() (devices []Gamepad) {
	return b.notifier.gamepads()
}

func (b *bus) Subscribe(id string) (err error) {
	if b.notifier == nil {
		return errors.New(ErrNotifierNotInitialized)
	}
	return b.notifier.subscribe(id)
}

func (b *bus) Unsubscribe(id string) (err error) {
	if b.notifier == nil {
		return errors.New(ErrNotifierNotInitialized)
	}
	return b.notifier.unsubscribe(id)
}

func (b *bus) Close() {

	if b.notifier == nil {
		return
	}

	err := b.notifier.stop()
	if err != nil {
		b.errChannel <- err
	}
	b.notifier = nil

	b.Lock()
	for _, ch := range b.channels {
		ch.CancelFunc()
	}
	b.Unlock()

	close(b.eventChannel)
	close(b.errChannel)

	return
}
