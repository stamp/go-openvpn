package openvpn

import (
	"bufio"
	"fmt"
	"os/exec"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/stamp/go-openssl"
)

type Process struct {
	StdOut     chan string `json:"-"`
	StdErr     chan string `json:"-"`
	Events     chan *Event `json:"-"`
	Stopped    chan bool   `json:"-"`
	parameters []string
	config     *Config
	Env        map[string]string
	Clients    map[string]*Client

	management *Management

	shutdown  chan bool
	waitGroup sync.WaitGroup
}

func NewProcess() *Process {
	p := &Process{
		Env:     make(map[string]string, 0),
		Events:  make(chan *Event, 10),
		Clients: make(map[string]*Client, 0),

		shutdown: make(chan bool),
	}

	p.management = NewManagement(p)

	return p
}

// Short-hands for some basic openvpn operating modes

func NewSslServer(ca *openssl.CA, cert *openssl.Cert, dh *openssl.DH, ta *openssl.TA) *Process { // {{{
	p := NewProcess()
	c := NewConfig()

	c.Device("tun")
	c.ServerMode(1194, ca, cert, dh, ta)
	c.IpPool("192.168.11.0/24")

	c.KeepAlive(10, 60)
	c.PingTimerRemote()
	c.PersistTun()
	c.PersistKey()

	p.SetConfig(c)
	return p
}                                                                                                               // }}}
func NewSslClient(remote string, ca *openssl.CA, cert *openssl.Cert, dh *openssl.DH, ta *openssl.TA) *Process { // {{{
	p := NewProcess()
	c := NewConfig()

	c.ClientMode(ca, cert, dh, ta)
	c.Remote(remote, 1194)
	c.Device("tun")

	c.KeepAlive(10, 60)
	c.PingTimerRemote()
	c.PersistTun()
	c.PersistKey()

	p.SetConfig(c)
	return p
}                                              // }}}
func NewStaticKeyServer(key string) *Process { // {{{
	p := NewProcess()
	c := NewConfig()

	c.Device("tun")
	c.IpConfig("10.8.0.1", "10.8.0.2")
	c.Secret(key)

	c.KeepAlive(10, 60)
	c.PingTimerRemote()
	c.PersistTun()
	c.PersistKey()

	p.SetConfig(c)
	return p
}                                                      // }}}
func NewStaticKeyClient(remote, key string) *Process { // {{{
	p := NewProcess()
	c := NewConfig()

	c.Remote(remote, 1194)
	c.Device("tun")
	c.IpConfig("10.8.0.2", "10.8.0.1")
	c.Secret(key)

	c.KeepAlive(10, 60)
	c.PingTimerRemote()
	c.PersistTun()
	c.PersistKey()

	p.SetConfig(c)
	return p
} // }}}

func (p *Process) SetConfig(c *Config) {
	p.config = c
}

func (p *Process) Start() (err error) { // {{{
	// Check if the process is already running
	if p.Stopped != nil {
		select {
		case <-p.Stopped:
			// Everything is good, no process running
		default:
			return fmt.Errorf("Openvpn is already started, aborting")
		}
	}

	// Start the management interface (if it isnt already started)
	path, err := p.management.Start()
	if err != nil {
		return err
	}

	// Add the management interface path to the config
	p.config.setManagementPath(path)

	return p.Restart()
}                                      // }}}
func (p *Process) Stop() (err error) { // {{{
	close(p.shutdown)
	p.waitGroup.Wait()

	return
}                                          // }}}
func (p *Process) Shutdown() (err error) { // {{{
	p.Stop()
	p.management.Shutdown()

	return
}                                         // }}}
func (p *Process) Restart() (err error) { // {{{
	// Fetch the current config
	config, err := p.config.Validate()
	if err != nil {
		return err
	}

	// Create the command
	cmd := exec.Command("openvpn", config...)

	// Attatch monitors for stdout, stderr and exit
	release := make(chan bool)
	defer close(release)
	p.ProcessMonitor(cmd, release)

	// Try to start the process
	err = cmd.Start()
	if err != nil {
		return err
	}

	return
} // }}}

func (p *Process) Fire(name string, args ...string) { // {{{
	select {
	case p.Events <- &Event{
		Name: name,
		Args: args,
	}:
	default:
		log.Warn("Lost event: ", name, " args:", args)
	}
} // }}}

func (p *Process) ProcessMonitor(cmd *exec.Cmd, release chan bool) { // {{{

	p.stdoutMonitor(cmd)
	p.stderrMonitor(cmd)

	p.Stopped = make(chan bool)

	go func() {
		p.waitGroup.Add(1)
		defer p.waitGroup.Done()

		defer close(p.Stopped)

		// Watch if the process exits
		done := make(chan error)
		go func() {
			<-release // Wait for the process to start
			done <- cmd.Wait()
		}()

		// Wait for shutdown or exit
		select {
		case <-p.shutdown:
			// Kill the server
			if err := cmd.Process.Kill(); err != nil {
				return
			}
			err := <-done // allow goroutine to exit
			log.Error("process killed with error = ", err)
		case err := <-done:
			log.Error("process done with error = ", err)
			return
		}

	}()
}                                                // }}}
func (p *Process) stdoutMonitor(cmd *exec.Cmd) { // {{{
	stdout, _ := cmd.StdoutPipe()
	go func() {
		p.waitGroup.Add(1)
		defer p.waitGroup.Done()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case p.StdOut <- scanner.Text():
			default:
				log.Trace("OPENVPN stdout: ", scanner.Text())
			}

		}
		if err := scanner.Err(); err != nil {
			log.Warn("OPENVPN stdout: (failed to read: ", err, ")")
			return
		}
	}()
}                                                // }}}
func (p *Process) stderrMonitor(cmd *exec.Cmd) { // {{{
	stderr, _ := cmd.StderrPipe()
	go func() {
		p.waitGroup.Add(1)
		defer p.waitGroup.Done()

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case p.StdErr <- scanner.Text():
			default:
				log.Warn("OPENVPN stderr: ", scanner.Text())
			}
		}
		if err := scanner.Err(); err != nil {
			log.Warn("OPENVPN stderr: (failed to read ", err, ")")
			return
		}
	}()
} // }}}
