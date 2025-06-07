package main

import (
	"fmt"
	"github.com/doingharm/go-gamepad-bus"
	"log"
)

func filterConnections(e *gamepads.Event) bool {

	if e.Type == gamepads.ControlEventType {
		data := e.Data.(gamepads.ControlEvent)

		if data.Type == gamepads.Button {
			return true
		}
	}

	return false
}

func main() {

	// initialize the bus
	b, errCh, err := gamepads.New()
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer b.Close()

	// read error messages
	go func() {
		for err = range errCh {
			log.Println(err.Error())
		}
	}()

	// set up a new event channel to listen to all devices.
	if ch := b.NewEventChannel(); ch != nil {
		for event := range ch.Ch {
			// if event is ConnectEvent, subscribe to the gamepad to receive all ControlEvents
			if event.Type == gamepads.ConnectEventType {
				if err = b.Subscribe(event.Data.(gamepads.Gamepad).ID); err != nil {
					log.Println(err.Error())
				}
			}

			fmt.Println(*event)

		}
	}

}
