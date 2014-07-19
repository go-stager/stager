package stager

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Configuration holds the config for a stager instance.
// This is usually filled from JSON, most likely from calling ReadConfig.
type Configuration struct {
	DomainSuffix string   // The suffix of the domain we're serving for
	Listen       string   // a host:port combination to bind to
	BasePort     int      // The base port number to start at
	MaxInstances int      // No more than this many instances can be made
	ProxyFormat  string   // Format template for building the proxy. advanced usage.
	InitCommand  []string // The command we run to initialize backends
	IdleTime     string   // Time duration after which an idle process is killed. runs through time.ParseDuration.
}

// IdleTimeDuration gets the idle time as a time.Duration.
// It allows you to specify the idle time as a duration like "5m" or "300s"
func (c Configuration) IdleTimeDuration() time.Duration {
	d, err := time.ParseDuration(c.IdleTime)
	if err != nil {
		panic(err)
	}
	return d
}

// DefaultConf is configuration defaults.
// The defaults are used for any key where it is omitted from both JSON and
// command-line config in the default application.
var DefaultConf = Configuration{
	DomainSuffix: ".stager:8000",
	Listen:       "127.0.0.1:8000",
	BasePort:     4200,
	MaxInstances: 100,
	ProxyFormat:  "http://127.0.0.1:{{.Port}}",
	InitCommand:  []string{"bash", "stager_script.sh"},
	IdleTime:     "5m",
}

// ReadConfig gets from both cmdline and JSON returning a new Configuration.
func ReadConfig() *Configuration {
	// Set up the eventual output configuration
	conf := &Configuration{}
	copyConfig(DefaultConf, conf, false)

	// Read config from various sources.
	cmdConf := &Configuration{}
	copyConfig(DefaultConf, cmdConf, false)
	configFile := ParseCommandConfig(cmdConf)

	if configFile != "" {
		jsonConf := &Configuration{}
		ParseJSONConfig(jsonConf, configFile)
		copyConfig(*jsonConf, conf, false)
	}
	// Overwrite JSON with any command-line args.
	copyConfig(*cmdConf, conf, false)
	fmt.Printf("%+v", conf)
	return conf
}

// ParseCommandConfig will fill a config struct from command-line options.
// The returned value is the name of a JSON config file to parse.
func ParseCommandConfig(config *Configuration) string {
	configFile := flag.String("config", "", "JSON Config file to parse")
	flag.StringVar(&config.Listen, "listen", config.Listen, "Listen on host:port")
	flag.StringVar(&config.ProxyFormat, "proxy_format", config.ProxyFormat, "Proxy template")
	flag.IntVar(&config.BasePort, "base_port", config.BasePort, "Base port num for instances")
	flag.IntVar(&config.MaxInstances, "max_instances", config.MaxInstances, "Maximum Instances")
	flag.StringVar(&config.IdleTime, "idle_time", config.IdleTime, "Idle time (duration)")
	flag.Var((*commandValue)(&config.InitCommand), "init_command", "Command to run to start instance")

	// set the values back to things we can tell are blank before parsing.
	copyConfig(Configuration{}, config, true)
	flag.Parse()

	return *configFile
}

// ParseJSONConfig will fill a config struct from a JSON config file.
func ParseJSONConfig(config *Configuration, configFile string) {
	file, _ := os.Open(configFile)
	decoder := json.NewDecoder(file)
	err := decoder.Decode(config)
	if err != nil {
		fmt.Println("error:", err)
	}
}

func copyConfig(src Configuration, dest *Configuration, force bool) {
	copyConfigString(src.DomainSuffix, &dest.DomainSuffix, force)
	copyConfigString(src.Listen, &dest.Listen, force)
	copyConfigInt(src.BasePort, &dest.BasePort, force)
	copyConfigInt(src.MaxInstances, &dest.MaxInstances, force)
	copyConfigString(src.ProxyFormat, &dest.ProxyFormat, force)
	copyConfigString(src.IdleTime, &dest.IdleTime, force)
	if force || len(src.InitCommand) > 0 {
		dest.InitCommand = src.InitCommand
	}
}

func copyConfigString(src string, dest *string, force bool) {
	if force || src != "" {
		*dest = src
	}
}

func copyConfigInt(src int, dest *int, force bool) {
	if force || src != 0 {
		*dest = src
	}
}

// commandValue implements the Value interface.
type commandValue []string

func (c *commandValue) String() string {
	return strconv.Quote(strings.Join(*c, " "))
}

func (c *commandValue) Set(s string) error {
	*c = strings.Split(s, " ")
	return nil
}
