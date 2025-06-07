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

type LinuxEventType uint8

const (
	IrrelevantEventType LinuxEventType = iota
	GamepadEventType
)

const (
	inputPath = "/dev/input"
)

type notifyLinux struct {
	sync.RWMutex
	verbose      bool
	ctx          context.Context
	cancelFunc   context.CancelFunc
	waitNotify   sync.WaitGroup
	gp           []*gamepadLinux
	eventChannel chan *Event
	errChannel   chan error
}

func linuxNotifier(eventChannel chan *Event, errChannel chan error, verbose bool) (nn notify, err error) {

	nl := &notifyLinux{
		eventChannel: eventChannel,
		ctx:          context.Background(),
		verbose:      verbose,
		errChannel:   errChannel,
	}

	nl.ctx, nl.cancelFunc = context.WithCancel(nl.ctx)

	var current []os.DirEntry
	current, err = os.ReadDir(inputPath)
	if err != nil {
		return
	}

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

		// Watch /dev for all events
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
		case GamepadEventType:
			nl.connectGamepad(name)
		default:
			return
		}
	case mask == unix.IN_DELETE:
		switch t {
		case GamepadEventType:
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
