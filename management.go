package openvpn

import "sync"

type Management struct {
	Conn            *Process
	ManagementRead  chan string `json:"-"`
	ManagementWrite chan string `json:"-"`

	Path string

	events chan []string

	currentClient string
	clientEnv     map[string]string

	buffer    []byte
	waitGroup sync.WaitGroup
	shutdown  chan bool
}

func NewManagement(conn *Process) *Management {
	return &Management{
		Conn:            conn,
		ManagementRead:  make(chan string),
		ManagementWrite: make(chan string),

		events: make(chan []string),

		clientEnv: make(map[string]string, 0),
		buffer:    make([]byte, 0),
		shutdown:  make(chan bool),
	}
}
