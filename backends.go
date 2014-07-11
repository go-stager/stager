package stager

import (
	"bytes"
	"errors"
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
	availPorts   []int
	proxyPrefix  *template.Template
	initCommand  []string
	notify       chan *Backend
}

func (m *backendManager) get(domain string) (b *Backend, err error) {
	name := domain[:len(domain)-m.suffixLength]
	b = m.backends[name]
	if b == nil {
		var port int
		port, err = m.allocatePort()
		if err != nil {
			return
		}
		m.Lock()
		b = &Backend{
			Name:   name,
			Port:   port,
			notify: m.notify,
		}
		m.backends[name] = b
		m.Unlock()

		buf := &bytes.Buffer{}
		err = m.proxyPrefix.Execute(buf, b)
		if err != nil {
			panic(err)
		}
		rawurl := string(buf.Bytes())
		u, err := url.Parse(rawurl)
		if err != nil {
			panic(err)
		}
		fmt.Printf("making new instance %s on port %d with backend url %s\n", name, b.Port, rawurl)

		b.proxy = httputil.NewSingleHostReverseProxy(u)

		go b.initialize(m.initCommand)

	}
	return
}

/* Allocate a port number to be used for another backend. */
func (m *backendManager) allocatePort() (int, error) {
	m.Lock()
	defer m.Unlock()
	l := len(m.availPorts)
	if l > 0 {
		port := m.availPorts[l-1]
		m.availPorts = m.availPorts[:l-1]
		return port, nil
	} else {
		return 0, errors.New("Not enough ports remain")
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
			backend.state = StateReaped
			fmt.Printf("Got state finished transition\n")
			m.Lock()
			delete(m.backends, backend.Name)
			m.Unlock()
			m.returnPort(backend.Port)
		} else {
			fmt.Printf("Backend %s, state %d\n", backend.Name, backend.state)
		}
	}
}

func newBackendManager(config *Configuration) *backendManager {
	// Make a slice of all available ports
	ports := make([]int, 0, config.MaxInstances)
	for i := config.BasePort + config.MaxInstances - 1; i >= config.BasePort; i-- {
		ports = append(ports, i)
	}

	manager := &backendManager{
		backends:     make(map[string]*Backend),
		suffixLength: len(config.DomainSuffix),
		availPorts:   ports,
		proxyPrefix:  template.Must(template.New("p").Parse(config.ProxyFormat)),
		initCommand:  config.InitCommand,
		notify:       make(chan *Backend),
	}
	go manager.watcher()
	return manager
}
