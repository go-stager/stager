package stager

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

type Configuration struct {
	DomainSuffix string // The suffix of the domain we're serving for
	Listen       string // a host:port combination to bind to
	BasePort     int    // The base port number to start at
	MaxInstances int    // No more than this many instances can be made
	ProxyFormat  string
	InitCommand  []string // The command we run to initialize backends
	IdleTime     string   // Time in seconds after which an idle process is killed.
}

func (c Configuration) IdleTimeDuration() time.Duration {
	d, err := time.ParseDuration(c.IdleTime)
	if err != nil {
		panic(err)
	}
	return d
}

func ReadConfig() *Configuration {
	config := &Configuration{}
	configFile := flag.String("config", "", "JSON Config file to parse")
	listen := flag.String("listen", "", "Listen on host:port")
	flag.StringVar(&config.ProxyFormat, "proxy_format", "http://127.0.0.1:{{.Port}}", "Proxy template")
	flag.IntVar(&config.BasePort, "base_port", 4200, "Base port num for instances")
	flag.IntVar(&config.MaxInstances, "max_instances", 100, "Maximum Instances")
	flag.StringVar(&config.IdleTime, "idle_time", "300s", "Idle time (duration)")

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
