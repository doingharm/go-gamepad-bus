package gamepads

import "context"

// EventChannel represents an event channel that can be used to receive events from gamepads.
type EventChannel struct {
	Ctx        context.Context
	Ch         chan *Event
	ErrCh      chan error
	CancelFunc context.CancelFunc
}

// FilterFunc is a function type used to filter events before they are sent to the event channel.
type FilterFunc func(e *Event) bool
