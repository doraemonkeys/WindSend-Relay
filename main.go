package main

import (
	"github.com/doraemonkeys/WindSend-Relay/config"
	"github.com/doraemonkeys/WindSend-Relay/global"
	"github.com/doraemonkeys/WindSend-Relay/relay"
	"go.uber.org/zap"
)

func init() {
	global.Init()
}

func main() {
	zap.L().Info("Relay server start")
	config := config.ParseConfig()
	relay := relay.NewRelay(*config)
	relay.Run()
}
