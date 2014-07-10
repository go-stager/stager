package main

import (
	"gopkg.in/stager.v0"
)

func main() {
	config := stager.ReadConfig()
	stager.Serve(config)
}
