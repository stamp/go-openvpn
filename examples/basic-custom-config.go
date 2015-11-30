package main

import (
	"log"

	"github.com/stamp/go-openvpn"
)

func main() {
	// A custom config example

	c := openvpn.NewConfig()
	c.Set("config", "myconfigfile.conf")

	// Create the openvpn instance
	p := openvpn.NewProcess()
	p.SetConfig(c)

	// Start the process
	p.Start()

	// Listen for events
	for {
		select {
		case event := <-p.Events:
			log.Println("Event: ", event.Name, "(", event.Args, ")")
		}
	}
}
