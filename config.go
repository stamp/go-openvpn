package openvpn

import (
	"net"
	"strconv"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/stamp/go-openssl"
)

type Config struct {
	flags  map[string]bool
	values map[string]string
	params []string
}

func NewConfig() *Config {
	return &Config{
		flags:  make(map[string]bool),
		values: make(map[string]string),
		params: make([]string, 0),
	}
}

func (c *Config) set(key, val string) {
	a := strings.Split("--"+key+" "+val, " ")
	for _, ar := range a {
		c.params = append(c.params, ar)
	}
}

func (c *Config) flag(key string) {
	//c.params = append(c.params, "--"+key)
	a := strings.Split("--"+key, " ")
	for _, ar := range a {
		c.params = append(c.params, ar)
	}
}

/*

17:18:12 Warn - openvpn.go - Current config: --management-client
17:18:12 Warn - openvpn.go - Current config: --management-hold
17:18:12 Warn - openvpn.go - Current config: --tls-server
17:18:12 Warn - openvpn.go - Current config: --ping-timer-rem
17:18:12 Warn - openvpn.go - Current config: --persist-key
17:18:12 Warn - openvpn.go - Current config: --management-signal
17:18:12 Warn - openvpn.go - Current config: --management-up-down
17:18:12 Warn - openvpn.go - Current config: --ifconfig-pool-linear
17:18:12 Warn - openvpn.go - Current config: --persist-tun
17:18:12 Warn - openvpn.go - Current config: --server 192.168.11.0/24
17:18:12 Warn - openvpn.go - Current config: --management /tmp/openvpn-management-12052.sock
17:18:12 Warn - openvpn.go - Current config: --mode server
17:18:12 Warn - openvpn.go - Current config: --key
17:18:12 Warn - openvpn.go - Current config: --port 1194
17:18:12 Warn - openvpn.go - Current config: --dev tun
17:18:12 Warn - openvpn.go - Current config: --ca
17:18:12 Warn - openvpn.go - Current config: --cert
17:18:12 Warn - openvpn.go - Current config: --tls-auth
17:18:12 Warn - openvpn.go - Current config: --keepalive 10 60

// "--verb 9", // Debug
X		"--mode server",
X		"--tls-server",
X		"--port 1194",
-		"--proto udp",
X		"--dev tun",
X		"--ca cert/CA/CA.crt",
X		"--cert cert/server/server.crt",
X		"--key cert/server/server.key",
		"--dh cert/DH1024.pem",
X		"--tls-auth cert/TA.key",
X		"--keepalive 5 10",     // Helper option for setting timeouts in server mode. Send ping once every n seconds, restart if ping not received for m seconds.
X		"--ping-timer-rem",     // Run the --ping-exit/--ping-restart timer only if we have a remote address.
X		"--persist-tun",        // On SIGUSR1 signals, remap signal (s='SIGHUP' or 'SIGTERM').
X		"--persist-key",        // Don't re-read key files across SIGUSR1 or --ping-restart.
X		"--management-hold",    // Start OpenVPN in a hibernating state, until a client of the management interface explicitly starts it.
X		"--management-signal",  // Issue SIGUSR1 when management disconnect event occurs.
X		"--management-up-down", // Report tunnel up/down events to management interface
		// "--management-client-auth" // Authenticate clients from management

X		"--server 192.168.11.0 255.255.255.0",
X		"--ifconfig-pool-linear",
		//"--ifconfig-pool 192.168.11.8 192.168.11.100",
-		"--client-to-client"


		// "--verb 9", // Debug
		"--client",
		"--remote "+server+" "+port,
		"--tls-client",
		"--proto udp",
		"--dev tun",
		"--ca cert/CA.crt",
		"--cert cert/client/client.crt",
		"--key cert/client/client.key",
		"--dh cert/DH1024.pem",
		"--tls-auth cert/TA.pem",
		//"--remote-cert-tls server",
		//"--ns-cert-type server",
		"--keepalive 10 60", // Helper option for setting timeouts in server mode. Send ping once every n seconds, restart if ping not received for m seconds.
		"--ping-timer-rem",  // Run the --ping-exit/--ping-restart timer only if we have a remote address.
		"--persist-tun",     // On SIGUSR1 signals, remap signal (s='SIGHUP' or 'SIGTERM').
		"--persist-key",     // Don't re-read key files across SIGUSR1 or --ping-restart.
		//"--management-hold",    // Start OpenVPN in a hibernating state, until a client of the management interface explicitly starts it.
		"--management-signal", // Issue SIGUSR1 when management disconnect event occurs.
		//"--management-up-down", // Report tunnel up/down events to management interface
		// "--management-client-auth" // Authenticate clients from management

*/

func (c *Config) Validate() (config []string, err error) {
	//for key, val := range c.values {
	//config = append(config, "--"+key+" "+val)
	//}

	//for key, _ := range c.flags {
	//config = append(config, "--"+key)
	//}

	return c.params, nil
}

func (c *Config) ServerMode(port int, ca *openssl.CA, cert *openssl.Cert, dh *openssl.DH, ta *openssl.TA) {
	c.set("mode", "server")
	c.set("port", strconv.Itoa(port))
	c.flag("tls-server")

	c.set("ca", ca.GetFilePath())
	c.set("crl-verify", ca.GetCRLPath())
	c.set("cert", cert.GetFilePath())
	c.set("key", cert.GetKeyPath())
	c.set("dh", dh.GetFilePath())
	c.set("tls-auth", ta.GetFilePath())
}
func (c *Config) ClientMode() {
	c.set("mode", "client")
}

func (c *Config) Remote(r string, port int) {
	c.set("remove", r)
}
func (c *Config) Protocol(p string) {
	c.set("proto", p)
}
func (c *Config) Device(t string) {
	c.set("dev", t)
}
func (c *Config) IpConfig(local, remote string) {
}
func (c *Config) IpPool(pool string) {

	ip, net, err := net.ParseCIDR(pool)
	if err != nil {
		log.Error(err)
		return
	}

	c.set("server", ip.String()+" "+strconv.Itoa(int(net.Mask[0]))+"."+strconv.Itoa(int(net.Mask[1]))+"."+strconv.Itoa(int(net.Mask[2]))+"."+strconv.Itoa(int(net.Mask[3])))
	//c.flag("ifconfig-pool-linear")
}

// openvpn --genkey --secret static.key
func (c *Config) Secret(key string) {
	c.set("secret", key)
}

func (c *Config) KeepAlive(interval, timeout int) {
	c.set("keepalive", strconv.Itoa(interval)+" "+strconv.Itoa(timeout))
}
func (c *Config) PingTimerRemote() {
	c.flag("ping-timer-rem")
}
func (c *Config) PersistTun() {
	c.flag("persist-tun")
}
func (c *Config) PersistKey() {
	c.flag("persist-key")
}

func (c *Config) Compression() {
	//comp-lzo
}
func (c *Config) ClientToClient() {
	c.flag("client-to-client")
}

func (c *Config) setManagementPath(path string) {
	c.set("management", path+" unix")
	c.flag("management-client")
	c.flag("management-hold")
	c.flag("management-signal")
	c.flag("management-up-down")

	log.Info("Current config:", c)
}
