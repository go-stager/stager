package stager

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Configuration struct {
	DomainSuffix string // The suffix of the domain we're serving for
	Listen       string // a host:port combination to bind to
	BasePort     int    // The base port number to start at
	MaxInstances int    // No more than this many instances can be made
}

func ReadConfig() *Configuration {
	config := &Configuration{}
	configFile := flag.String("config", "", "JSON Config file to parse")
	listen := flag.String("listen", "", "Listen on host:port")

	flag.Parse()

	if *configFile != "" {
		file, _ := os.Open(*configFile)
		decoder := json.NewDecoder(file)
		err := decoder.Decode(config)
		if err != nil {
			fmt.Println("error:", err)
		}
	}

	if *listen != "" {
		config.Listen = *listen
	}
	return config
}
