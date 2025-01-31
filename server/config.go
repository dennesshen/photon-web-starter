package server

import "github.com/dennesshen/photon-core-starter/configuration"

func init() {
	configuration.Register(&config)
}

type Config struct {
	Server struct {
		Port        string `mapstructure:"port"`
		Swagger     string `mapstructure:"swagger"`
		ContextPath string `mapstructure:"context-path"`
	} `mapstructure:"server"`
}

var config Config
