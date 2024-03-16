package config

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
)

type Config struct {
	Postgres  Postgres  `yaml:"postgres"`
	Transport Transport `yaml:"transport"`
}

type Postgres struct {
	Hostname string `yaml:"POSTGRES_HOSTNAME"`
	Port     string `yaml:"POSTGRES_PORT"`
	User     string `yaml:"POSTGRES_USER"`
	Password string `yaml:"POSTGRES_PASSWORD"`
	DBName   string `yaml:"POSTGRES_DB"`
}

type Transport struct {
	Hostname string `yaml:"hostname"`
	Port     string `yaml:"port"`
}

func MustLoad() *Config {
	cfg := &Config{}

	configFile, err := ioutil.ReadFile("config/config.yaml")
	if err != nil {
		log.Fatalf("Error while open file: %v", err)
	}

	err = yaml.Unmarshal(configFile, cfg)

	if err != nil {
		log.Fatalf("Error while reading env file: %v", err)
	}

	return cfg
}
