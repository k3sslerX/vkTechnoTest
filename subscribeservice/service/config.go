package service

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	GRPCPort string `envconfig:"SUBSSERVICE_GRPC_PORT" default:"8080"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process("", cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
