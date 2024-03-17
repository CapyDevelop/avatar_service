package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
)

type Config struct {
	Postgres  Postgres
	Transport Transport
}

type Postgres struct {
	Hostname string `env:"POSTGRES_HOSTNAME"`
	Port     string `env:"POSTGRES_PORT"`
	User     string `env:"POSTGRES_USER"`
	Password string `env:"POSTGRES_PASSWORD"`
	DBName   string `env:"POSTGRES_DB"`
}

type Transport struct {
	Hostname string `env:"hostname"`
	Port     string `env:"port"`
}

func MustLoad() *Config {
	//cfg := &Config{}
	//
	//configFile, err := ioutil.ReadFile("config/config.yaml")
	//if err != nil {
	//	log.Fatalf("Error while open file: %v", err)
	//}
	//
	//err = yaml.Unmarshal(configFile, cfg)
	//
	//if err != nil {
	//	log.Fatalf("Error while reading env file: %v", err)
	//}

	cfg := &Config{}

	err := cleanenv.ReadEnv(cfg)
	if err != nil {
		log.Fatalf("cannot read env: %v", err)
	}

	return cfg
}
