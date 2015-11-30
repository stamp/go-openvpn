go-openvpn
==========

A go library to start and interface with openvpn processes. 

### Basic static key example
First use the following command to create a PSK (pre shared key)

    openvpn --genkey --secret pre-shared.key

#### Server
    // Create an instance of the openvpn struct
  	p := openvpn.NewStaticKeyServer("pre-shared.key")
  
    // Start the openvpn process. Note that this method do not block so the program will continue at once.
  	p.Start()
  
    // Listen for events
  	for {
  		select {
  		case event := <-p.Events:
  			log.Println("Event: ", event.Name, "(", event.Args, ")")
  		}
  	}
	
#### Client
    // Create an instance of the openvpn struct
  	p := openvpn.NewStaticKeyClient("localhost", "pre-shared.key")
  	
    // Start the openvpn process. Note that this method do not block so the program will continue at once.
  	p.Start()
  
    // Listen for events
  	for {
  		select {
  		case event := <-p.Events:
  			log.Println("Event: ", event.Name, "(", event.Args, ")")
  		}
  	}
