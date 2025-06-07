package gamepads

// Gamepad holds information of a gamepad
type Gamepad struct {
	ID        string
	Model     string
	Buttons   int
	ButtonMap []int
	Axes      int
	AxesMap   []int
}
