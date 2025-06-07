package gamepads

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	gpName       = 0x80006a13 + (128 << 16)
	gpAxes       = 0x80016a11 /* get number of axes */
	gpButtons    = 0x80016a12
	gpVersion    = 0x80046a01
	gpAxesMap    = 0x80406a32
	gpButtonsMap = 0x80406a34
	// gpCorrectionValues = 0x80406a22
)

type gamepadLinux struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	file       *os.File
	id         string
	path       string
	devName    string
	buttons    uint8
	buttonsMap [768]uint16
	axes       uint8
	axesMap    [64]uint8
	version    int32
	subscribed bool
}

type eventLinux struct {
	Timestamp uint32
	Value     int16
	Type      uint8
	Index     uint8
}

func newLinuxGamepad(name, path string) (*gamepadLinux, error) {

	f, err := openFilePersistent(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	gp := &gamepadLinux{
		id:   name,
		path: path,
		file: f,
	}

	if err = ioctlStr(f, gpName, &gp.devName); err != nil {
		return nil, err
	}
	if err = ioctl(f, gpButtons, unsafe.Pointer(&gp.buttons)); err != nil {
		return nil, err
	}
	if err = ioctl(f, gpAxes, unsafe.Pointer(&gp.axes)); err != nil {
		return nil, err
	}
	if err = ioctl(f, gpVersion, unsafe.Pointer(&gp.version)); err != nil {
		return nil, err
	}
	if err = ioctl(f, gpButtonsMap, unsafe.Pointer(&gp.buttonsMap)); err != nil {
		return nil, err
	}
	if err = ioctl(f, gpAxesMap, unsafe.Pointer(&gp.axesMap)); err != nil {
		return nil, err
	}

	return gp, nil
}

func (g *gamepadLinux) subscribe(eventChannel chan *Event) (err error) {

	if g.subscribed {
		return errors.New(ErrJoystickAlreadySubscribed)
	}

	if g.file, err = openFilePersistent(g.path); err != nil {
		return
	}

	g.ctx, g.cancelFunc = context.WithCancel(context.Background())
	g.subscribed = true

	go func() {
		id := g.id
		for {

			var e eventLinux
			if binary.Read(g.file, binary.LittleEndian, &e) != nil {
				return
			}

			select {
			case <-g.ctx.Done():
				_ = g.file.Close()
				return
			default:
			}

			eventChannel <- &Event{
				Type: ControlEventType,
				ID:   id,
				Data: ControlEvent{
					Timestamp: e.Timestamp,
					Type:      ControlType(e.Type),
					Index:     int(e.Index),
					Value:     e.Value,
				},
			}
		}
	}()

	return
}

func (g *gamepadLinux) unsubscribe() (err error) {

	if !g.subscribed {
		return errors.New(ErrJoystickAlreadyUnsubscribed)
	}

	g.cancelFunc()
	g.subscribed = false
	return
}

func ioctl(f *os.File, infoType int, dest unsafe.Pointer) (err error) {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		f.Fd(),
		uintptr(infoType),
		uintptr(dest),
	)
	if errno != 0 {
		return fmt.Errorf("ioctl error: %d", errno)
	}
	return
}

func ioctlStr(f *os.File, infoType int, dest *string) (err error) {
	info := make([]byte, 128)
	if err = ioctl(f, infoType, unsafe.Pointer(&info[0])); err != nil {
		return
	}
	*dest = escapeString(info)
	return
}
