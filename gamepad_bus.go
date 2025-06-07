package gamepads

import (
	"context"
	"errors"
	"runtime"
	"sync"
)

type bus struct {
	sync.RWMutex
	verbose      bool
	eventChannel chan *Event
	errChannel   chan error
	notifier     notify
	channels     []*EventChannel
}

type Bus interface {
	NewEventChannel(filters ...FilterFunc) (dest *EventChannel)
	Gamepads() (gamepads []Gamepad)
	Subscribe(id string) (err error)
	Unsubscribe(id string) (err error)
	Close()
}

func New(verbose bool) (b Bus, errCh <-chan error, err error) {

	dest := &bus{
		verbose:      verbose,
		eventChannel: make(chan *Event),
		errChannel:   make(chan error),
	}

	switch runtime.GOOS {
	case "linux":
		if dest.notifier, err = linuxNotifier(dest.eventChannel, dest.errChannel, false); err != nil {
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
