package stager

import (
	"fmt"
	"net/http/httputil"
	"net/url"
)

type backend struct {
	proxy *httputil.ReverseProxy
}

type backendManager struct {
	backends     map[string]*backend
	suffixLength int
}

func (m backendManager) get(domain string) *backend {
	suffix := domain[:-m.suffixLength]
	u, _ := url.Parse("http://www.example.com/" + suffix)
	if m.backends[suffix] == nil {
		fmt.Println("making new suffix %s", suffix)
		m.backends[suffix] = &backend{
			proxy: httputil.NewSingleHostReverseProxy(u),
		}
	}
	return m.backends[suffix]
}

func newBackendManager(config *Configuration) backendManager {
	manager := backendManager{
		backends:     *new(map[string]*backend),
		suffixLength: len(config.DomainSuffix),
	}
	return manager
}
