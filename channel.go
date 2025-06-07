package gamepads

import "context"

type EventChannel struct {
	Ctx        context.Context
	Ch         chan *Event
	ErrCh      chan error
	CancelFunc context.CancelFunc
}

type FilterFunc func(e *Event) bool
