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
)

// A backend represents one possible instance we can be proxying to.
// The public properties Port and Name can be used as configuration data.
type Backend struct {
	Port    int
	Name    string
	proxy   *httputil.ReverseProxy
	running bool
	command *exec.Cmd
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
	b.running = true
	go b.waiter()
}

// waiter runs in a goroutine waiting for process to end.
func (b *Backend) waiter() {
	b.command.Wait()
	b.command = nil
	b.running = false
	fmt.Printf("Backend %s on port %d exited.\n", b.Name, b.Port)
}

// backendManager manages backends, allocating ports and backends as needed.
// Use the function newBackendManager to initialize properly.
type backendManager struct {
	sync.Mutex
	backends     map[string]*Backend
	suffixLength int
	currentPort  int
	availPorts   []int
	proxyPrefix  *template.Template
	initCommand  []string
}

func (m *backendManager) get(domain string) *Backend {
	name := domain[:len(domain)-m.suffixLength]
	if m.backends[name] == nil {
		port := m.allocatePort()
		b := &Backend{
			Port: port,
			Name: name,
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

/* Return a port number to the available ports slice */
func (m *backendManager) returnPort(portNum int) {
	m.Lock()
	m.availPorts = append(m.availPorts, portNum)
	m.Unlock()
}

func newBackendManager(config *Configuration) *backendManager {
	manager := &backendManager{
		backends:     make(map[string]*Backend),
		suffixLength: len(config.DomainSuffix),
		currentPort:  config.BasePort,
		proxyPrefix:  template.Must(template.New("p").Parse(config.ProxyFormat)),
		initCommand:  config.InitCommand,
	}
	return manager
}
