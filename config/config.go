package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/doraemonkeys/WindSend-Relay/version"
	"go.uber.org/zap"
)

type SecretInfo struct {
	SecretKey string `json:"secret_key" env:"KEY,notEmpty"`
	MaxConn   int    `json:"max_conn" env:"MAX_CONN" envDefault:"5"`
}

type Config struct {
	ListenAddr  string       `json:"listen_addr" env:"WS_LISTEN_ADDR,notEmpty" envDefault:"0.0.0.0:16779"`
	MaxConn     int          `json:"max_conn" env:"WS_MAX_CONN" envDefault:"100"`
	IDWhitelist []string     `json:"id_whitelist" envPrefix:"WS_ID_WHITELIST"`
	SecretInfo  []SecretInfo `json:"secret_info" envPrefix:"WS_SECRET"`
	NoAuth      bool         `json:"no_auth" env:"WS_NO_AUTH" envDefault:"false"`
	// LogLevel    string       `json:"log_level" env:"WS_LOG_LEVEL" envDefault:"INFO"`
}

func ParseConfig() *Config {
	configFile := flag.String("config", "", "json config file, other command line args will be ignored")
	useEnv := flag.Bool("use-env", false, "use env, other command line args will be ignored")

	var config Config
	flag.StringVar(&config.ListenAddr, "listen-addr", "0.0.0.0:16779", "listen address")
	flag.IntVar(&config.MaxConn, "max-conn", 100, "max connection")
	flag.BoolVar(&config.NoAuth, "no-auth", false, "allow all connections")
	// flag.StringVar(&config.LogLevel, "log-level", "INFO", "log level")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Println("WindSend-Relay", "v"+version.Version)
		fmt.Println("BuildTime:", version.BuildTime)
		fmt.Println("BuildHash:", version.BuildHash)
		os.Exit(0)
	}

	if *useEnv {
		return parseEnv()
	}

	if *configFile != "" {
		jsonFile, err := os.Open(*configFile)
		if err != nil {
			zap.L().Error("Failed to open config file", zap.Error(err))
		}
		defer jsonFile.Close()

		json.NewDecoder(jsonFile).Decode(&config)
	}

	return &config
}

func parseEnv() *Config {
	var config, err = env.ParseAs[Config]()
	if err != nil {
		zap.L().Error("Failed to parse env", zap.Error(err))
	}
	return &config
}
