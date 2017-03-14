package openvpn

import (
	"encoding/json"
	"errors"
	log "github.com/cihub/seelog"
	"github.com/stamp/go-openssl"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	remote string
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

/**
Loads configuration from the given file, A  configuration is only valid if:
1. It is a string- In this case, append a # symbol at the end to ignore
2. Array of flags-- In this case, append # symbol at the end of the flag to ignore it
3. Array of push params-- Append # as above to ignore a push
*/
func (c *Config) LoadFile(filename string) error {
	cfgFile, err := os.Open(filename)
	if err != nil {
		return errors.New("File " + filename + " could not be read: " + err.Error())
	}
	var cc interface{}
	loader := json.NewDecoder(cfgFile)
	errd := loader.Decode(&cc)
	if errd != nil {
		return errors.New("Could not decode JSON file: " + errd.Error())
	}
	jmap := cc.(map[string]interface{})
	for k, v := range jmap {
		switch vu := v.(type) {
		case string:
			s := v.(string)
			if !strings.HasSuffix(s, "#") {
				c.Set(k, s)
			}
		case []interface{}:
			if k == "flags" {
				for _, vv := range vu {
					if !strings.HasSuffix(vv.(string), "#") {
						c.Flag(vv.(string))
					}
				}
			} else {
				for _, vv := range vu {
					if !strings.HasSuffix(vv.(string), "#") {
						c.Flag(k + " \"" + vv.(string) + "\"")
					}
				}
			}
		default:
			log.Debug("Not valid variable", k, ":", v)
		}

	}
	return nil
}

func (c *Config) Refresh() {
	c.params = c.params[0:0] //Clear the array first
	for key, val := range c.values {
		a := strings.Split("--"+key+" "+val, " ")
		for _, ar := range a {
			c.params = append(c.params, ar)
		}
	}
	for key, val := range c.flags {
		if val {
			a := strings.Split("--"+key, " ")
			for _, er := range a {
				c.params = append(c.params, er)
			}
		}
	}
}

/**
If called for a second time replace the key
*/
func (c *Config) Set(key, val string) {
	if v, ok := c.values[key]; ok {
		if v == val { //Skip if already set
			return
		}
		delete(c.values, key)
	}
	c.values[key] = val
	c.Refresh()
}

func (c *Config) Flag(key string) {
	if _, ok := c.flags[key]; ok { //Skip if already set
		return
	}
	c.flags[key] = true
	c.Refresh()
}

func (c *Config) Validate() (config []string, err error) {
	return c.params, nil
}

func (c *Config) ServerMode(port int, ca *openssl.CA, cert *openssl.Cert, dh *openssl.DH, ta *openssl.TA) {
	c.Set("mode", "server")
	c.Set("port", strconv.Itoa(port))
	c.Flag("tls-server")

	c.Set("ca", ca.GetFilePath())
	c.Set("crl-verify", ca.GetCRLPath())
	c.Set("cert", cert.GetFilePath())
	c.Set("key", cert.GetKeyPath())
	c.Set("dh", dh.GetFilePath())
	c.Set("tls-auth", ta.GetFilePath()+" 0")
}
func (c *Config) ClientMode(ca *openssl.CA, cert *openssl.Cert, dh *openssl.DH, ta *openssl.TA) {
	c.Flag("client")
	c.Flag("tls-client")

	c.Set("ca", ca.GetFilePath())
	c.Set("cert", cert.GetFilePath())
	c.Set("key", cert.GetKeyPath())
	c.Set("dh", dh.GetFilePath())
	c.Set("tls-auth", ta.GetFilePath()+" 1")
}

func (c *Config) Remote(r string, port int) {
	c.Set("port", strconv.Itoa(port))
	c.Set("remote", r)
	c.remote = r
}
func (c *Config) Protocol(p string) {
	c.Set("proto", p)
}
func (c *Config) Device(t string) {
	c.Set("dev", t)
}
func (c *Config) IpConfig(local, remote string) {
}
func (c *Config) IpPool(pool string) {

	ip, net, err := net.ParseCIDR(pool)
	if err != nil {
		log.Error(err)
		return
	}

	c.Set("server", ip.String()+" "+strconv.Itoa(int(net.Mask[0]))+"."+strconv.Itoa(int(net.Mask[1]))+"."+strconv.Itoa(int(net.Mask[2]))+"."+strconv.Itoa(int(net.Mask[3])))
}

func (c *Config) Secret(key string) {
	c.Set("secret", key)
}

func (c *Config) KeepAlive(interval, timeout int) {
	c.Set("keepalive", strconv.Itoa(interval)+" "+strconv.Itoa(timeout))
}
func (c *Config) PingTimerRemote() {
	c.Flag("ping-timer-rem")
}
func (c *Config) PersistTun() {
	c.Flag("persist-tun")
}
func (c *Config) PersistKey() {
	c.Flag("persist-key")
}

func (c *Config) Compression() {
	//comp-lzo
}
func (c *Config) ClientToClient() {
	c.Flag("client-to-client")
}

func (c *Config) setManagementPath(path string) {
	if path != "" {
		c.Set("management", path+" unix")
		c.Flag("management-client")
		c.Flag("management-hold")
		c.Flag("management-signal")
		c.Flag("management-up-down")

		log.Info("Current config:", c)
	}
}
