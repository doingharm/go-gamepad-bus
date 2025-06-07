package gamepads

import (
	"errors"
	"os"
	"time"
)

func escapeString(src []byte) string {
	n := 0
	for _, b := range src {
		if b != 0 {
			src[n] = b
			n++
		}
	}
	return string(src[:n])
}

func openFilePersistent(path string) (f *os.File, err error) {

	for i := 0; i < 5; i++ {
		if f, err = os.OpenFile(path, os.O_RDWR, 0); err != nil {
			if errors.Is(err, os.ErrPermission) {
				if i == 4 {
					return
				}
				timer := time.NewTimer(200 * time.Millisecond)
				<-timer.C
				timer.Stop()
				continue
			} else {
				return
			}
		}
		break
	}
	return
}
