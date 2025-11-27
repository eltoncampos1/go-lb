package config

import (
	"flag"
	"log"
)

type Config struct {
	Port        int
	ServerList  string
	HealthCheckInterval int
}

func Load() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerList, "backends", "", "Load balanced backends, use commas to separate")
	flag.IntVar(&cfg.Port, "port", 3030, "Port to serve")
	flag.IntVar(&cfg.HealthCheckInterval, "health-check-interval", 20, "Health check interval in seconds")
	flag.Parse()

	if len(cfg.ServerList) == 0 {
		log.Fatal("Please provide one or more backends to load balance")
	}

	return cfg
}
