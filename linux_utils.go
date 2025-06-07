package gamepads

import "bytes"

func extractFromBytes(src []byte) (t linuxEventType, name string, ok bool) {
	switch {
	case bytes.Compare(src[:2], []byte{106, 115}) == 0:
		return gamepadEventType, escapeString(src), true
	default:
		return irrelevantEventType, "", false
	}
}

func parseButtonsMap(mp [768]uint16, count int) (dest []int) {

	if count == 0 {
		return
	}

	for i, m := range mp {
		if i != 0 && m == 0 {
			continue
		}
		dest = append(dest, int(m))
	}
	return
}

func parseAxesMap(mp [64]uint8, count int) (dest []int) {

	if count == 0 {
		return
	}

	for i, m := range mp {
		if i != 0 && m == 0 {
			continue
		}
		dest = append(dest, int(m))
	}
	return
}
