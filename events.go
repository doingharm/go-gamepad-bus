package gamepads

// EventType represents the different types of events that can occur.
type EventType uint8

const (
	ConnectEventType EventType = iota
	DisconnectEventType
	ControlEventType
)

// Event represents a single event that can be sent on the channel.
type Event struct {
	Type EventType
	ID   string
	Data any
}

// ControlType represents the different types of control events.
type ControlType uint8

const (
	Button       ControlType = 0x01
	Axes         ControlType = 0x02
	InitialState ControlType = 0x08
)

// ControlEvent represents a single control event, such as a button press or axis movement.
type ControlEvent struct {
	Timestamp uint32
	Type      ControlType
	Index     int
	Value     int16
}
