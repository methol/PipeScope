package main

import (
	"flag"
	"fmt"
	"log"

	"pipescope/internal/config"
)

func main() {
	configPath := flag.String("config", "assets/config.example.yaml", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	fmt.Printf("pipescope admin listen at %s:%d\n", cfg.Admin.Host, cfg.Admin.Port)
}

