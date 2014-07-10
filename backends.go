package stager

import (
	"bytes"
	"fmt"
	"net/http/httputil"
	"net/url"
	"sync"
	"text/template"
)

type Backend struct {
	Port    int
	Name    string
	proxy   *httputil.ReverseProxy
	running bool
}

// This will do the setup of the backend at some point.
func (b *Backend) initialize() {
	b.running = true
}

type backendManager struct {
	backends     map[string]*Backend
	suffixLength int
	currentPort  int
	availPorts   []int
	lockPorts    sync.Mutex
	proxyPrefix  *template.Template
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

		go b.initialize()

	}
	return m.backends[name]
}

/* Allocate a port number to be used for another backend. */
func (m *backendManager) allocatePort() int {
	m.lockPorts.Lock()
	defer m.lockPorts.Unlock()
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
	m.lockPorts.Lock()
	m.availPorts = append(m.availPorts, portNum)
	m.lockPorts.Unlock()
}

func newBackendManager(config *Configuration) *backendManager {
	manager := &backendManager{
		backends:     make(map[string]*Backend),
		suffixLength: len(config.DomainSuffix),
		currentPort:  config.BasePort,
		proxyPrefix:  template.Must(template.New("p").Parse(config.ProxyFormat)),
	}
	return manager
}
