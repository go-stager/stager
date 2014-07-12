package stager

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
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
	Port    int       // The TCP port this instance uses.
	Name    string    // The name or key of this instance.
	LastReq time.Time // Last time this instance was requested
	url     *url.URL
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
	go b.startCheck()
	go b.waiter()
}

func (b *Backend) transition(state State) {
	b.state = state
	b.notify <- b
}

// startCheck makes sure we started properly.
// It does this by sending a HEAD request periodically until we get a
// response that's not a 5xx response.
func (b *Backend) startCheck() {
	time.Sleep(BackendCheckDelay)
	for a := 0; b.state == StateStarted && a < BackendCheckAttempts; a++ {
		resp, err := http.Head(b.url.String())
		if err != nil {
			fmt.Printf("Backend %s didn't connect yet\n", b.Name)
		} else if resp.StatusCode >= 500 {
			fmt.Printf("Backend %s got a >5xx status code\n", b.Name)
		} else {
			b.transition(StateRunning)
		}
		time.Sleep(BackendCheckDelay)
	}
}

// waiter runs in a goroutine waiting for process to end.
func (b *Backend) waiter() {
	err := b.command.Wait()
	if err != nil {
		b.transition(StateErrored)
		time.Sleep(5 * time.Second)
	}
	fmt.Printf("Backend %s on port %d exited.\n", b.Name, b.Port)
	b.command = nil
	b.transition(StateFinished)
}

// kill the running process
func (b *Backend) kill() {
	if b.state == StateStarted || b.state == StateRunning {
		proc := b.command.Process
		if proc != nil {
			proc.Signal(os.Interrupt)
		}
	}
}

// backendManager manages backends, allocating ports and backends as needed.
// Use the function newBackendManager to initialize properly.
type backendManager struct {
	sync.Mutex
	config       *Configuration
	backends     map[string]*Backend
	suffixLength int
	availPorts   []int
	proxyPrefix  *template.Template
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
			Name:    name,
			Port:    port,
			LastReq: time.Now(),
			notify:  m.notify,
		}
		m.backends[name] = b
		m.Unlock()

		buf := &bytes.Buffer{}
		err = m.proxyPrefix.Execute(buf, b)
		if err != nil {
			return
		}
		rawurl := string(buf.Bytes())
		b.url, err = url.Parse(rawurl)
		if err != nil {
			return
		}
		fmt.Printf("making new instance %s on port %d with backend url %s\n", name, b.Port, rawurl)

		b.proxy = httputil.NewSingleHostReverseProxy(b.url)

		go b.initialize(m.config.InitCommand)

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
	tick := time.Tick(BackendIdleCheck)
	for {
		select {
		case backend := <-m.notify:
			if backend.state == StateFinished {
				backend.state = StateReaped
				fmt.Printf("Backend %s, state finished\n", backend.Name)
				m.Lock()
				delete(m.backends, backend.Name)
				m.Unlock()
				m.returnPort(backend.Port)
			} else {
				fmt.Printf("Backend %s, state %d\n", backend.Name, backend.state)
			}
		case <-tick:
			threshold := m.config.IdleTimeDuration()
			m.Lock()
			for _, backend := range m.backends {
				if time.Since(backend.LastReq) > threshold {
					fmt.Printf("Killing idle worker %s\n", backend.Name)
					go backend.kill()
				}
			}
			m.Unlock()
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
		config:       config,
		backends:     make(map[string]*Backend),
		suffixLength: len(config.DomainSuffix),
		availPorts:   ports,
		proxyPrefix:  template.Must(template.New("p").Parse(config.ProxyFormat)),
		notify:       make(chan *Backend),
	}
	go manager.watcher()
	return manager
}
