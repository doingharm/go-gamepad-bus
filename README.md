# 🎮 go-gamepad-bus

A high-performance, event-driven **gamepad/joystick bus** written in pure Go.

This module uses the `notify` functionality from the [`golang.org/x/sys`](https://pkg.go.dev/golang.org/x/sys) package to monitor device connections and disconnections in **real-time**. It operates through scalable, **customizable event channels** and a **clean subscription interface**, enabling responsive and modular input handling.

## 🧩 Built for Stream Integrations

Designed with **stream-based communication** in mind, it fits naturally into single-stream connections such as WebSockets, gRPC streams, or WebRTC data channels, making it ideal for webview and networked applications.

`gamepadbus` is ideal for:

- WebSockets
- gRPC streams
- WebRTC data channels
- UDP/TCP connections
- or any (unary) stream connection where live gamepad data needs to flow.

## 🐧 Platform Support

Currently, this module is **Linux-only**.

I built it to scratch my own itch, generally i don't publish my own packages/applications but I felt like trying it out — existing packages lacked the hot-reloading and interface style I truly wanted. I don’t intend to add support for Windows or macOS, but the architecture was designed to allow easy extension to other platforms. I might intend to add some tests later on. 

**Feel free to contribute support for other operating systems.**

## 🚀 Features

- Real-time detection of device connect/disconnect
- Non-blocking, channel-based event model
- Simple subscription mechanism
- Designed for seamless integration with streaming protocols

### Example
```
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

```