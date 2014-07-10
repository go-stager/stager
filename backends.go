package stager

import (
	"fmt"
	"net/http/httputil"
	"net/url"
)

type backend struct {
	port    int
	proxy   *httputil.ReverseProxy
	running bool
}

// This will do the setup of the backend at some point.
func (b *backend) initialize() {

}

type backendManager struct {
	backends     map[string]*backend
	suffixLength int
	currentPort  int
	availPorts   []int
}

func (m *backendManager) get(domain string) *backend {
	name := domain[:len(domain)-m.suffixLength]
	if m.backends[name] == nil {
		port := m.allocatePort()
		fmt.Printf("making new instance %s on port %d\n", name, port)
		u, _ := url.Parse("http://www.example.com/" + name)
		b := m.backends[name] = &backend{
			port:  port,
			proxy: httputil.NewSingleHostReverseProxy(u),
		}
		go b.initialize()

	}
	return m.backends[name]
}

// Ick, this is very old school and probably not concurrency friendly. But I need something now.
func (m *backendManager) allocatePort() int {
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

func newBackendManager(config *Configuration) *backendManager {
	manager := &backendManager{
		backends:     make(map[string]*backend),
		suffixLength: len(config.DomainSuffix),
		currentPort:  config.BasePort,
	}
	return manager
}
