package main

import (
	"log"

	"github.com/stamp/go-openvpn"
)

func main() {
	// A basic pre-shared-static-key openvpn tunnel

	// First generate a static key using:
	// openvpn --genkey --secret pre-shared.key
	// and distribute to both client and server

	// Create the openvpn instance
	p := openvpn.NewStaticKeyServer("pre-shared.key")

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
