package openvpn

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"net"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
)

var uniqId = 0

func (m *Management) Start() (path string, err error) { // {{{
	m.Path = "/tmp/openvpn-management-" + strconv.Itoa(uniqId) + ".sock"
	uniqId += 1

	log.Info("Management start: unix:" + m.Path)

	// Open the socket
	l, err := net.Listen("unix", m.Path)
	if err != nil {
		log.Error(err)
		return
	}

	// Wait for shutdown
	go func() {
		<-m.shutdown
		l.Close()
	}()

	// Wait for connections
	go func() {
		m.waitGroup.Add(1)
		defer m.waitGroup.Done()

		for {
			fd, err := l.Accept()
			log.Info("Management: openvpn management interface have connected")
			if err != nil {
				select {
				case <-m.shutdown:
					log.Info("Management: closed")
				default:
					log.Critical("accept error:", err)
				}
				return
			}

			go m.server(fd)
		}
	}()

	return m.Path, err
} // }}}

func (m *Management) Shutdown() {
	log.Info("Management: shutdown")
	close(m.shutdown)

	m.waitGroup.Wait()
	log.Info("Management: shutdown done")
}

func (m *Management) server(c net.Conn) { // {{{

	go func() {
		//c.Write([]byte("status\n"))

		if m.Conn.config.remote == "" {
			log.Info("Management started in SERVER mode")
			t := time.NewTicker( time.Second * 10)
			for {
				select {
				case <-m.shutdown:
					t.Stop()
					log.Info("Management: closed")
					return
				case <-t.C:
					c.Write([]byte("status\n"))
				}
			}
		} else {
			log.Info("Management started in CLIENT mode")
		}
		/*
			for {
				<-time.After(time.Second)
				_, err := c.Write([]byte("push \"echo hej\"\n"))
				if err != nil {
					log.Error("Failed management write: ", err)
					return
				}
				log.Info("push echo hej")
			}*/
	}()

	go func() {
		for {
			select {
			case <-m.shutdown:
				log.Info("Management: closed")
				return
			case rows := <-m.events:
				m.route(c, rows[0], rows[1:])
			}
		}
	}()

	reader := bufio.NewReader(c)
	tp := textproto.NewReader(reader)
	for {
		select {
		case <-m.shutdown:
			log.Info("Management: closed")
			return
		case <- time.After( 1 * time.Second):
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			m.parse([]byte(line), false)
		}
	}
} // }}}

func (m *Management) parse(line []byte, retry bool) { // {{{
	//log.Error("Parse: ", string(line))

	types := map[string]string{
		"client-list": "(?ms)OpenVPN CLIENT LIST\n" +
			"Updated,([^\n]*)\n" +
			"(.*)\n" +
			"ROUTING TABLE\n" +
			"(.*)\n" +
			"GLOBAL STATS\n" +
			"(.*)" +
			"\nEND\n",

		"log":      ">LOG:([^\r\n]*)$",  // -- Log message output as controlled by the "log" command.
		"info":     ">INFO:([^\r\n]*)$", // -- Informational messages such as the welcome message.
		"error":    "ERROR:([^\r\n]*)$",
		"fatal":    "FATAL:([^\r\n]*)$",  // -- A fatal error which is output to the log file just prior to OpenVPN exiting.
		"hold":     ">HOLD:([^\r\n]*)$",  // -- Used to indicate that OpenVPN is in a holding state and will not start until it receives a "hold release" command.
		"state":    ">STATE:([^\r\n]*)$", // -- Show the current OpenVPN state, show state history, or enable real-time notification of state changes.
		"success":  "SUCCESS: ([^\r\n]*)$",
		"updown":   ">UPDOWN:([^=,\r\n]+),([^=\r\n]+)=([^\r\n]+)$",
		"updown-1": ">UPDOWN:([^=\r\n]+)$",

		//BYTECOUNT -- Real-time bandwidth usage notification, as enabled
		//by "bytecount" command when OpenVPN is running as
		//a client.

		//BYTECOUNT_CLI -- Real-time bandwidth usage notification per-client,
		//as enabled by "bytecount" command when OpenVPN is
		//running as a server.

		//ECHO     -- Echo messages as controlled by the "echo" command.

		//NEED-OK  -- OpenVPN needs the end user to do something, such as
		//insert a cryptographic token.  The "needok" command can
		//be used to tell OpenVPN to continue.

		//NEED-STR -- OpenVPN needs information from end, such as
		//a certificate to use.  The "needstr" command can
		//be used to tell OpenVPN to continue.

		//PASSWORD -- Used to tell the management client that OpenVPN
		//needs a password, also to indicate password
		//verification failure.

		//STATE    -- Shows the current OpenVPN state, as controlled
		//by the "state" command.

		//CID --	Client ID, numerical ID for each connecting client, sequence = 0,1,2,...
		//KID --	Key ID, numerical ID for the key associated with a given client TLS session,
		//			sequence = 0,1,2,...
		//PRI --	Primary (1) or Secondary (0) VPN address/subnet.  All clients have at least
		//			one primary IP address.  Secondary address/subnets are associated with;
		//			client-specific "iroute" directives.
		//ADDR --	IPv4 address/subnet in the form 1.2.3.4 or 1.2.3.0/255.255.255.0
		"client-connect":     ">CLIENT:CONNECT,([\\d]+),([\\d]+)",          // Notify new client connection {CID},{KID}
		"client-reauth":      ">CLIENT:REAUTH,([\\d]+),([\\d]+)",           // existing client TLS session renegotiation {CID}, {KID}
		"client-established": ">CLIENT:ESTABLISHED,([\\d]+)",               // Notify successful client authentication and session initiation {CID}
		"client-disconnect":  ">CLIENT:DISCONNECT,([\\d]+)",                // Notify existing client disconnection {CID}
		"client-address":     ">CLIENT:ADDRESS,([\\d]+),([\\d]+),([\\d]+)", //Notify that a particular virtual address or subnet is now associated with a specific client. {CID},{ADDR},{PRI}
		"client-env":         ">CLIENT:ENV,([^=\r\n]+)=([^\r\n]*)",
		"client-end":         ">CLIENT:ENV,END",
	}

mainLoop:
	for t, r := range types {
		reg, _ := regexp.Compile(r)
		match := reg.FindAllSubmatchIndex(line, -1)
		if len(match) == 0 {
			continue
		}

		for _, row := range match {
			// Extract all strings of the current match
			strings := []string{t}
			for index := range row {
				if index%2 > 0 { // Skipp all odd indexes
					continue
				}

				strings = append(strings, string(line[row[index]:row[index+1]]))
			}

			// Try to deliver the message
			select {
			case m.events <- strings:
			case <-time.After(time.Second):
				log.Errorf("Failed to transport message (%p): %s |%s|", m.events, t, row, strings)
			}

			if row[0] > 0 {
				log.Warn("Trowing away message: ", strconv.Quote(string(line[:row[0]])))
			}

			// Just save the rest of the message
			line = bytes.Trim(line[row[1]:], "\x00")

			continue mainLoop
		}
	}

	if len(line) > 0 && !retry {
		//log.Warn("Could not find message, adding to buffer: ", string(line))

		m.buffer = append(m.buffer, line...)
		m.buffer = append(m.buffer, '\n')
		m.parse(m.buffer, true)
	} else if len(line) > 0 {
		m.buffer = line
	}

	//log.Error("Buffer: ", string(m.buffer))
} // }}}

func (m *Management) route(c net.Conn, t string, row []string) { // {{{
	switch t {
	case "log":
		log.Trace(row[1])
	case "info":
		log.Info(row[1])
	case "error":
		log.Error(row[1])
	case "fatal":
		log.Critical(row[1])
	case "hold":
		log.Info("HOLD active:", row[1])

		c.Write([]byte("echo on\n"))
		c.Write([]byte("state on\n"))
		c.Write([]byte("hold release\n"))
	case "state":
		state := strings.Split(row[1], ",")
		if len(state) < 2 {
			log.Error("Failed to decode state:", state)
			return
		}

		log.Info("STATE:", state[1])

		switch state[1] {
		case "CONNECTING":
		case "RESOLVE":
		case "WAIT":
		case "AUTH":
		case "GET_CONFIG":
		case "ASSIGN_IP":
		case "ADD_ROUTES":
		case "CONNECTED":
			m.Conn.Fire("Connected", state[3])
		case "RECONNECTING":
			m.Conn.Fire("Disconnected")
		case "EXITING":
			m.Conn.Fire("Disconnected")
		default:
			log.Error("Recived unkown state:", state[1])
		}
	case "success":
		//log.Info(row[1]);
	case "client-list":
		m.clientList(row)
	case "updown":
		m.Conn.Env[row[2]] = row[3]
	case "client-connect", "client-reauth":
		m.currentClient = row[1]
		m.clientEnv = make(map[string]string, 0)
	case "client-established":
		m.currentClient = row[1]
		m.clientEnv = make(map[string]string, 0)
	case "client-disconnected":
		m.currentClient = row[1]
		m.clientEnv = make(map[string]string, 0)
	case "client-env":
		if m.clientEnv != nil {
			m.clientEnv[row[1]] = row[2]
		} else {
			log.Error("Throwing away ENV data: ", row[1], "=", row[2])
		}
	case "client-end":

		// Check if the CN is set
		if cn, ok := m.clientEnv["X509_0_CN"]; ok {
			// Check if there is a connected client with that CN
			if _, ok := m.Conn.Clients[cn]; !ok {
				//log.Info("Adding new client: ", cn)
				m.Conn.Clients[cn] = &Client{
					CommonName:   cn,
					PublicIP:     "",
					BytesRecived: "0",
					BytesSent:    "0",
					LastRef:      "0",
				}
				m.Conn.Fire("client connected", cn)

				//go m.Conn.clientWorker(cn)

			}

			if m.Conn.Clients[cn].Env == nil {
				m.Conn.Clients[cn].Env = make(map[string]string, 0)
			}

			for key, val := range m.clientEnv {
				m.Conn.Clients[cn].Env[key] = val
				//log.Error(key, " := ", val)
			}
			m.Conn.Fire("client updated", cn)
		}
	case "client-address":
		m.currentClient = row[1]
		m.clientEnv = make(map[string]string, 0)
	default:
		log.Error(t, ": ", row[1:])
	}

} // }}}

func (m *Management) clientList(match []string) { // {{{
	if len(match) < 3 {
		log.Error("Invalid client list, regexp failed: ", match)
		return
	}

	clients := makeCsvList(match[2])
	routes := makeCsvList(match[3])
	//stats := makeCsvList(match[4])

	var checked map[string]*Client

	Clone(m.Conn.Clients, &checked)

	for _, c := range clients {
		//log.Info("Client: ", index, ": ", c)
		if _, ok := m.Conn.Clients[c["Common Name"]]; ok {
			delete(checked, c["Common Name"]) // Remove from checked
			m.Conn.Clients[c["Common Name"]].missing = 0
			m.Conn.Clients[c["Common Name"]].PublicIP = c["Real Address"]
			m.Conn.Clients[c["Common Name"]].BytesRecived = c["Bytes Received"]
			m.Conn.Clients[c["Common Name"]].BytesSent = c["Bytes Sent"]
			m.Conn.Clients[c["Common Name"]].LastRef = c["Last Ref"]
		} else {
			//log.Info("Adding new client: ", c["Common Name"]);
			m.Conn.Clients[c["Common Name"]] = &Client{
				CommonName:   c["Common Name"],
				PublicIP:     c["Real Address"],
				BytesRecived: c["Bytes Received"],
				BytesSent:    c["Bytes Sent"],
				LastRef:      c["Last Ref"],
			}

			m.Conn.Fire("client connected", c["Common Name"])

			//go m.Conn.clientWorker(c["Common Name"], block)
		}
	}
	for _, c := range routes {
		//log.Info("Route: ", index, ": ", c)
		if _, ok := m.Conn.Clients[c["Common Name"]]; ok {
			m.Conn.Clients[c["Common Name"]].PrivateIP = c["Virtual Address"]

			if m.Conn.Clients[c["Common Name"]].waitForPrivateIP != nil {
				close(m.Conn.Clients[c["Common Name"]].waitForPrivateIP)
				m.Conn.Clients[c["Common Name"]].waitForPrivateIP = nil
			}
		}
	}

	// Remove all clients that isnt connected
	for index := range checked {
		m.Conn.Clients[index].missing++

		if m.Conn.Clients[index].missing > 5 {
			//log.Info("Removing disconnected client from list: ", index)
			delete(m.Conn.Clients, index)

			m.Conn.Fire("client removed", index)
		}
	}

	//for row, text := range stats {
	//log.Info("Stat: ", row, ": ", text)
	//}
	//Common Name,Real Address,Bytes Received,Bytes Sent,Connected Since
	//VPN_client,10.13.156.4:1194,12563,14885,Thu Feb 13 23:39:20 2014

	//Virtual Address,Common Name,Real Address,Last Ref
	//192.168.11.4,VPN_client,10.13.156.4:1194,Thu Feb 13 23:39:20 2014

	//Max bcast/mcast queue length,0
} // }}}
func makeCsvList(data string) (list []map[string]string) { // {{{
	list = make([]map[string]string, 0)

	rows := strings.Split(data, "\n")

	cols := strings.Split(rows[0], ",")

	for i, row := range rows[1:] {
		values := strings.Split(row, ",")

		list = append(list, make(map[string]string, 0))

		for c, col := range cols {
			list[i][col] = values[c]
		}
	}
	return
} // }}}

func Clone(a, b interface{}) { // {{{

	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	dec := gob.NewDecoder(buff)
	enc.Encode(a)
	dec.Decode(b)
} // }}}
