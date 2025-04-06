package main

import (
	"github.com/doraemonkeys/WindSend-Relay/config"
	"github.com/doraemonkeys/WindSend-Relay/global"
	"github.com/doraemonkeys/WindSend-Relay/relay"
)

func main() {
	config := config.ParseConfig()

	global.InitLogger(config.LogLevel)

	relay := relay.NewRelay(*config)
	relay.Run()
}
