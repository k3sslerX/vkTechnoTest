package service

import (
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
	"time"
)

type Config struct {
	GRPCPort string        `envconfig:"SUBSSERVICE_GRPC_PORT" default:":1313"`
	Timeout  time.Duration `envconfig:"SUBSSERVICE_TIMEOUT" default:"5s"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process("", cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewLogger() *log.Logger {
	return log.New(os.Stdout, "[pubsub] ", log.LstdFlags|log.Lshortfile)
}
