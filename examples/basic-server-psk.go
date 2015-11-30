package main

import (
	"log"

	"github.com/stamp/go-openvpn"
)

func main() {

	// First generate a static key using:
	// openvpn --genkey --secret pre-shared.key
	// and distribute to both client and server

	p := openvpn.NewStaticKeyServer("pre-shared.key")

	p.Start()

	for {
		select {
		case event := <-p.Events:
			log.Println("Event: ", event.Name, "(", event.Args, ")")
		}
	}
}
