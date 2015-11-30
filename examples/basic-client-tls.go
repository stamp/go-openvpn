package main

import (
	"log"

	"github.com/stamp/go-openssl"
	"github.com/stamp/go-openvpn"
)

func main() {

	// First generate a static key using:
	// openvpn --genkey --secret pre-shared.key
	// and distribute to both client and server

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
		CommonName:   "Example client",
		Email:        "Example email",
	}

	if ca, err = ssl.LoadOrCreateCA("ca.crt", "ca.key"); err != nil {
		log.Println("LoadOrCreateCA failed: ", err)
		return
	}
	// Note the last bool parameter! This is important beacuse it will generate a "client"-cert
	if cert, err = ssl.LoadOrCreateCert("clients/client1.crt", "clients/client1.key", "client1", ca, false); err != nil {
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

	p := openvpn.NewSslClient("localhost", ca, cert, dh, ta)

	p.Start()

	for {
		select {
		case event := <-p.Events:
			log.Println("Event: ", event.Name, "(", event.Args, ")")
		}
	}
}
