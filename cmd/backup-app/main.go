package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	//Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Configuration load error: %v", err)
	}

	fmt.Printf("Configuration loaded:\n")
	fmt.Printf(" Server Port: %d\n", cfg.ServerPort)
	fmt.Printf(" Backup Directory: %s\n", cfg.BackupDir)
	fmt.Printf(" Path to DataBase: %s\n", cfg.DatabasePath)
	fmt.Printf(" Secret salt (first 5 symbols): %s...\n", cfg.SecretSalt[:5])

	fmt.Println("\nApplication successfully loaded. Server will be run next...")
}

type Config struct {
	ServerPort   int    `yaml:"server_port"`
	BackupDir    string `yaml:"backup_directory"`
	DatabasePath string `yaml:"database_path"`
	SecretSalt   string `yaml:"secret_salt"`
}

func LoadConfig() (*Config, error) {
	configPath := filepath.Join("configs", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config file read error %s: %w", configPath, err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("config file parsing error with %s: %w", configPath, err)
	}

	return &cfg, nil
}
