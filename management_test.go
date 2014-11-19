package openvpn

import (
	"strconv"
	"sync"
	"testing"
)

var waitGroup sync.WaitGroup

func init() {
	//logger, err := log.LoggerFromConfigAsFile("logconfig.xml")
	//if err != nil {
	//testConfig := `
	//<seelog type="sync">
	//<outputs formatid="main">
	//<filter levels="trace">
	//<console formatid="colored-trace"/>
	//</filter>
	//<filter levels="debug">
	//<console formatid="colored-debug"/>
	//</filter>
	//<filter levels="info">
	//<console formatid="colored-info"/>
	//</filter>
	//<filter levels="warn">
	//<console formatid="colored-warn"/>
	//</filter>
	//<filter levels="error">
	//<console formatid="colored-error"/>
	//</filter>
	//<filter levels="critical">
	//<console formatid="colored-critical"/>
	//</filter>
	//</outputs>
	//<formats>
	//<format id="colored-trace"  format="%Time %EscM(40)%Level%EscM(49) - %File:%Line - %Msg%n%EscM(0)"/>
	//<format id="colored-debug"  format="%Time %EscM(45)%Level%EscM(49) - %File:%Line - %Msg%n%EscM(0)"/>
	//<format id="colored-info"  format="%Time %EscM(46)%Level%EscM(49) - %File:%Line - %Msg%n%EscM(0)"/>
	//<format id="colored-warn"  format="%Time %EscM(43)%Level%EscM(49) - %File:%Line - %Msg%n%EscM(0)"/>
	//<format id="colored-error"  format="%Time %EscM(41)%Level%EscM(49) - %File:%Line - %Msg%n%EscM(0)"/>
	//<format id="colored-critical"  format="%Time %EscM(41)%Level%EscM(49) - %File:%Line - %Msg%n%EscM(0)"/>
	//</formats>
	//</seelog>`

	//logger, _ = log.LoggerFromConfigAsBytes([]byte(testConfig))
	//}
	//log.ReplaceLogger(logger)
}

func TestNewManagement(t *testing.T) {

	m := NewManagement(&Process{})

	if m == nil {
		t.Error("Return is nil")
	}
}

func TestParse(t *testing.T) {
	m := &Management{
		events: make(chan []string),
		buffer: make([]byte, 0),
	}

	done := make(chan bool)
	go func() {
		waitGroup.Add(1)
		defer waitGroup.Done()

		select {
		case result := <-m.events:
			for index := range result {
				t.Log("Result[", index, "]: \n", strconv.Quote(result[index]))
			}

			if len(result) != 6 {
				t.Error("Wrong length on answer, should be 5, is ", len(result))
			}

			if result[0] != "client-list" {
				t.Error("result[0] is invalid")
			}
			if result[1] != "OpenVPN CLIENT LIST\n"+
				"Updated, Thu Feb 13 23:39:20 2014\n"+
				"Common Name,Real Address,Bytes Received,Bytes Sent,Connected Since\n"+
				"VPN_client,10.13.156.4:1194,12563,14885,Thu Feb 13 23:39:20 2014\n"+
				"ROUTING TABLE\n"+
				"Virtual Address,Common Name,Real Address,Last Ref\n"+
				"192.168.11.4,VPN_client,10.13.156.4:1194,Thu Feb 13 23:39:20 2014\n"+
				"GLOBAL STATS\n"+
				"Max bcast/mcast queue length,0\n"+
				"END\n" {
				t.Error("result[1] is invalid")
			}
			if result[2] != " Thu Feb 13 23:39:20 2014" {
				t.Error("result[2] is invalid")
			}
			if result[3] !=
				"Common Name,Real Address,Bytes Received,Bytes Sent,Connected Since\n"+
					"VPN_client,10.13.156.4:1194,12563,14885,Thu Feb 13 23:39:20 2014" {
				t.Error("result[3] is invalid")
			}
			if result[4] !=
				"Virtual Address,Common Name,Real Address,Last Ref\n"+
					"192.168.11.4,VPN_client,10.13.156.4:1194,Thu Feb 13 23:39:20 2014" {
				t.Error("result[4] is invalid")
			}
			if result[5] != "Max bcast/mcast queue length,0" {
				t.Error("result[5] is invalid")
			}

			return
		case <-done:
			t.Error("Parse done without result")
		}
	}()

	m.parse([]byte("OpenVPN CLIENT LIST"), false)
	m.parse([]byte("Updated, Thu Feb 13 23:39:20 2014"), false)
	m.parse([]byte("Common Name,Real Address,Bytes Received,Bytes Sent,Connected Since"), false)
	m.parse([]byte("VPN_client,10.13.156.4:1194,12563,14885,Thu Feb 13 23:39:20 2014"), false)
	m.parse([]byte(""), false)
	m.parse([]byte("ROUTING TABLE"), false)
	m.parse([]byte("Virtual Address,Common Name,Real Address,Last Ref"), false)
	m.parse([]byte("192.168.11.4,VPN_client,10.13.156.4:1194,Thu Feb 13 23:39:20 2014"), false)
	m.parse([]byte(""), false)
	m.parse([]byte("GLOBAL STATS"), false)
	m.parse([]byte("Max bcast/mcast queue length,0"), false)
	m.parse([]byte("END"), false)

	close(done)

	waitGroup.Wait()
}
