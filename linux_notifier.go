package gamepads

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

// linuxEventType is an enumeration of possible event types on a Linux system.
type linuxEventType uint8

const (
	irrelevantEventType linuxEventType = iota
	gamepadEventType
)

const (
	inputPath = "/dev/input"
)

type notifyLinux struct {
	sync.RWMutex
	ctx          context.Context
	cancelFunc   context.CancelFunc
	waitNotify   sync.WaitGroup
	gp           []*gamepadLinux
	eventChannel chan *Event
	errChannel   chan error
}

// linuxNotifier creates a Linux-specific gamepad notification system.
func linuxNotifier(eventChannel chan *Event, errChannel chan error) (nn notify, err error) {

	nl := &notifyLinux{
		eventChannel: eventChannel,
		ctx:          context.Background(),
		errChannel:   errChannel,
	}

	// Create a new context with a cancel function for stopping the notification system.
	nl.ctx, nl.cancelFunc = context.WithCancel(nl.ctx)

	var current []os.DirEntry
	current, err = os.ReadDir(inputPath)
	if err != nil {
		return
	}

	// Create a new goroutine for each device to watch for create events (i.e., when the gamepad is connected).
	for _, entry := range current {
		nl.waitNotify.Add(1)
		go nl.handleEvent(unix.IN_CREATE, []byte(entry.Name()))
	}

	go func() {
		nl.waitNotify.Wait()

		var fd int
		fd, err = unix.InotifyInit()
		if err != nil {
			nl.errChannel <- fmt.Errorf("inotify init failed: %v", err)
			return
		}
		defer func() {
			err = syscall.Close(fd)
			nl.errChannel <- fmt.Errorf("syscall close failed: %v", err)
			return
		}()

		var wd int
		wd, err = unix.InotifyAddWatch(fd, inputPath, unix.IN_CREATE|unix.IN_DELETE|unix.IN_MODIFY)
		if err != nil {
			nl.errChannel <- fmt.Errorf("inotify add watch failed: %v", err)
			return
		}
		defer func() {
			if _, err = unix.InotifyRmWatch(fd, uint32(wd)); err != nil {
				nl.errChannel <- fmt.Errorf("inotify remove watch failed: %v", err)
				return
			}
		}()

		buf := make([]byte, 512)

		for {

			select {
			case <-nl.ctx.Done():
				return
			default:
			}
			var n int
			n, err = syscall.Read(fd, buf)
			if err != nil {
				nl.errChannel <- fmt.Errorf("read failed: %v", err)
				return
			}

			var offset uint32
			for offset <= uint32(n-unix.SizeofInotifyEvent) {
				event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
				nameBytes := buf[offset+unix.SizeofInotifyEvent : offset+unix.SizeofInotifyEvent+uint32(event.Len)]
				nl.waitNotify.Add(1)
				nl.handleEvent(int(event.Mask), nameBytes)
				offset += unix.SizeofInotifyEvent + uint32(event.Len)
			}
			nl.waitNotify.Wait()
		}
	}()

	return nl, nil
}

// gamepads returns a list of connected gamepads.
func (nl *notifyLinux) gamepads() (devices []Gamepad) {
	nl.RLock()
	defer nl.RUnlock()
	for _, j := range nl.gp {
		devices = append(devices, Gamepad{
			ID:        j.id,
			Model:     j.devName,
			Buttons:   int(j.buttons),
			ButtonMap: parseButtonsMap(j.buttonsMap, int(j.buttons)),
			Axes:      int(j.axes),
			AxesMap:   parseAxesMap(j.axesMap, int(j.axes)),
		})
	}
	return
}

// stop stops the notification system.
func (nl *notifyLinux) stop() (err error) {

	nl.RLock()
	defer nl.RUnlock()

	for _, gp := range nl.gp {
		if err = gp.unsubscribe(); err != nil {
			nl.errChannel <- err
		}
	}

	nl.cancelFunc()
	return
}

// subscribe subscribes to the gamepad with the given ID.
func (nl *notifyLinux) subscribe(id string) (err error) {
	nl.RLock()
	defer nl.RUnlock()

	for _, gp := range nl.gp {
		if gp.id != id {
			continue
		}

		return gp.subscribe(nl.eventChannel)

	}

	return fmt.Errorf(ErrJoystickNotFound, id)
}

// unsubscribe unsubscribes from the gamepad with the given ID.
func (nl *notifyLinux) unsubscribe(id string) (err error) {
	nl.RLock()
	defer nl.RUnlock()

	for _, gp := range nl.gp {
		if gp.id != id {
			continue
		}

		return gp.unsubscribe()

	}

	return fmt.Errorf(ErrJoystickNotFound, id)
}

// handleEvent is called when a new event is received from the inotify system.
func (nl *notifyLinux) handleEvent(mask int, bt []byte) {

	defer nl.waitNotify.Done()

	t, name, ok := extractFromBytes(bt)

	if !ok {
		return
	}

	// Decode flags
	switch {
	case mask == unix.IN_CREATE:

		switch t {
		case gamepadEventType:
			nl.connectGamepad(name)
		default:
			return
		}
	case mask == unix.IN_DELETE:
		switch t {
		case gamepadEventType:
			nl.eventChannel <- &Event{
				Type: DisconnectEventType,
				ID:   name,
				Data: nil,
			}
		default:
			return
		}
	default:
	}
}

// connectGamepad is called when a new gamepad device is connected.
func (nl *notifyLinux) connectGamepad(name string) {

	path := fmt.Sprintf("%s/%s", inputPath, name)

	newGp, err := newLinuxGamepad(name, path)
	if err != nil {
		nl.errChannel <- err
		return
	}

	nl.Lock()
	nl.gp = append(nl.gp, newGp)
	nl.Unlock()

	select {
	case <-nl.ctx.Done():
		return
	default:
		nl.eventChannel <- &Event{
			Type: ConnectEventType,
			ID:   newGp.id,
			Data: Gamepad{
				ID:        newGp.id,
				Model:     newGp.devName,
				Buttons:   int(newGp.buttons),
				ButtonMap: parseButtonsMap(newGp.buttonsMap, int(newGp.buttons)),
				Axes:      int(newGp.axes),
				AxesMap:   parseAxesMap(newGp.axesMap, int(newGp.axes)),
			},
		}
	}

}
