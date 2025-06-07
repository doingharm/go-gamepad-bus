package gamepads

type notify interface {
	stop() (err error)
	gamepads() (devices []Gamepad)
	subscribe(id string) (err error)
	unsubscribe(id string) (err error)
}
