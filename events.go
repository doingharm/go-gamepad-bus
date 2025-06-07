package gamepads

type EventType uint8

const (
	ConnectEventType EventType = iota
	DisconnectEventType
	ControlEventType
)

type Event struct {
	Type EventType
	ID   string
	Data any
}

type ControlType uint8

const (
	Button       ControlType = 0x01
	Axes         ControlType = 0x02
	InitialState ControlType = 0x08
)

type ControlEvent struct {
	Timestamp uint32
	Type      ControlType
	Index     int
	Value     int16
}
