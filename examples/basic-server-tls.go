package main

import (
	"log"

	"github.com/stamp/go-openssl"
	"github.com/stamp/go-openvpn"
)

func main() {
	// This example first tries to load and if not found creates all the components needed for a TLS tunnel

	var err error
	var ca *openssl.CA
	var cert *openssl.Cert
	var dh *openssl.DH
	var ta *openssl.TA

	ssl := openssl.Openssl{
		Path: "certs", // A storage folder, where to store all certs

		Country:      "SE",
		Province:     "Example provice",
		City:         "Example city",
		Organization: "Example organization",
		CommonName:   "Example commonname",
		Email:        "Example email",
	}

	if ca, err = ssl.LoadOrCreateCA("ca.crt", "ca.key"); err != nil {
		log.Println("LoadOrCreateCA failed: ", err)
		return
	}
	// Note the last bool parameter! This is important beacuse it will generate a "server"-cert
	if cert, err = ssl.LoadOrCreateCert("server/server.crt", "server/server.key", "server", ca, true); err != nil {
		log.Println("LoadOrCreateCert failed: ", err)
		return
	}
	if dh, err = ssl.LoadOrCreateDH("DH1024.pem", 1024); err != nil {
		log.Println("LoadOrCreateDH failed: ", err)
		return
	}
	if ta, err = ssl.LoadOrCreateTA("TA.key"); err != nil {
		log.Println("LoadOrCreateTA failed: ", err)
		return
	}

	// Create the openvpn instance
	p := openvpn.NewSslServer(ca, cert, dh, ta)

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
