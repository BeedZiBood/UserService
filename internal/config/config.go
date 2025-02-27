package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env           string  `yaml:"env" env:"ENV"`
	Sources       uint    `yaml:"sources" env:"SOURCES"`
	FlowIntensity float64 `yaml:"flow_intensity" env:"FLOW_INTENSITY"`
	HTTPServer    `yaml:"http_server"`
	KafkaProducer `yaml:"kafka_producer"`
}

type HTTPServer struct {
	Address     string        `yaml:"address"`
	Timeout     time.Duration `yaml:"timeout"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

type KafkaProducer struct {
	Broker string `yaml:"broker"`
	Topic  string `yaml:"topic"`
}

func MustLoad() Config {
	configPath := fetchConfigPath()

	if configPath == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file %s doesn't exist", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("connot read config: %s", err)
	}

	return cfg
}

func fetchConfigPath() string {
	var result string

	flag.StringVar(&result, "config", "", "path to the config")
	flag.Parse()

	if result == "" {
		result = os.Getenv("CONFIG_PATH")
	}

	return result
}
