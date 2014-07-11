package stager

import (
	"bytes"
	"fmt"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"text/template"
	"time"
)

// A backend represents one possible instance we can be proxying to.
// The public properties Port and Name can be used as configuration data.
type Backend struct {
	Port    int
	Name    string
	proxy   *httputil.ReverseProxy
	state   State
	command *exec.Cmd
	notify  chan *Backend
}

// initialize starts the backend running.
func (b *Backend) initialize(command []string) {
	// setup prerequisite vars
	environ := os.Environ()
	environ = append(environ, fmt.Sprintf("STAGER_PORT=%d", b.Port), fmt.Sprintf("STAGER_NAME=%s", b.Name))

	// Build the command
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = environ
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	b.command = cmd
	b.transition(StateStarted)
	go b.waiter()
}

func (b *Backend) transition(state State) {
	b.state = state
	b.notify <- b
}

// waiter runs in a goroutine waiting for process to end.
func (b *Backend) waiter() {
	// This is a little hack to temporarily flip the running state.
	time.Sleep(10 * time.Second)
	b.transition(StateRunning)
	b.command.Wait()
	b.transition(StateFinished)
	b.command = nil
	fmt.Printf("Backend %s on port %d exited.\n", b.Name, b.Port)
}

// backendManager manages backends, allocating ports and backends as needed.
// Use the function newBackendManager to initialize properly.
type backendManager struct {
	sync.Mutex
	backends     map[string]*Backend
	suffixLength int
	currentPort  int
	maxPort      int
	availPorts   []int
	proxyPrefix  *template.Template
	initCommand  []string
	notify       chan *Backend
}

func (m *backendManager) get(domain string) *Backend {
	name := domain[:len(domain)-m.suffixLength]
	if m.backends[name] == nil {
		port := m.allocatePort()
		b := &Backend{
			Port:   port,
			Name:   name,
			notify: m.notify,
		}
		m.backends[name] = b
		buf := &bytes.Buffer{}
		err := m.proxyPrefix.Execute(buf, b)
		if err != nil {
			panic(err)
		}
		rawurl := string(buf.Bytes())
		u, err := url.Parse(rawurl)
		if err != nil {
			panic(err)
		}
		fmt.Printf("making new instance %s on port %d with backend url %s\n", name, port, rawurl)

		b.proxy = httputil.NewSingleHostReverseProxy(u)

		go b.initialize(m.initCommand)

	}
	return m.backends[name]
}

/* Allocate a port number to be used for another backend. */
func (m *backendManager) allocatePort() int {
	m.Lock()
	defer m.Unlock()
	l := len(m.availPorts)
	if l > 0 {
		port := m.availPorts[l-1]
		m.availPorts = m.availPorts[:l-1]
		return port
	} else {
		port := m.currentPort
		m.currentPort += 1
		return port
	}
}

// Return a port number to the available ports slice
func (m *backendManager) returnPort(portNum int) {
	m.Lock()
	m.availPorts = append(m.availPorts, portNum)
	m.Unlock()
}

// watcher watches on the channel for things which happen
func (m *backendManager) watcher() {
	for backend := range m.notify {
		if backend.state == StateFinished {
			fmt.Printf("Got state finished transition")
			m.Lock()
			delete(m.backends, backend.Name)
			m.Unlock()
			m.returnPort(backend.Port)
		} else {
			fmt.Printf("Backend %s, state %d", backend.Name, backend.state)
		}
	}
}

func newBackendManager(config *Configuration) *backendManager {
	manager := &backendManager{
		backends:     make(map[string]*Backend),
		suffixLength: len(config.DomainSuffix),
		currentPort:  config.BasePort,
		maxPort:      config.BasePort + config.MaxInstances,
		proxyPrefix:  template.Must(template.New("p").Parse(config.ProxyFormat)),
		initCommand:  config.InitCommand,
		notify:       make(chan *Backend),
	}
	go manager.watcher()
	return manager
}
