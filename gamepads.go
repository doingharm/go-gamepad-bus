package gamepads

type Gamepad struct {
	ID        string
	Model     string
	Buttons   int
	ButtonMap []int
	Axes      int
	AxesMap   []int
}
