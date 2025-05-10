package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	GRPCPort        int           `yaml:"GRPCPort"`
	ShutdownTimeout time.Duration `yaml:"ShutdownTimeout"`
}

func New(path string) (*Config, error) {
	cfg := Config{}
	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
