package main

import (
	"backup-app/internal/backup"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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
	fmt.Printf(" Server Timeouts: Read=%s, Write=%s, Idle=%s\n", cfg.ReadTimeout, cfg.WriteTimeout, cfg.IdleTimeout)
	fmt.Printf(" Shutdown Timeout: %s\n", cfg.ShutdownTimeout)

	//--- Backup Test ---
	sourceFile := "E:\\test.txt"

	destFile := filepath.Join(cfg.BackupDir, "test_backup", filepath.Base(sourceFile))

	log.Printf("Backup try: %s -> %s\n", sourceFile, destFile)
	err = backup.CopyFiles(sourceFile, destFile)
	if err != nil {
		log.Printf("Local backup error: %v", err)
	} else {
		log.Printf("Local backup successfully completed for test file.")
	}

	//--- End of Backup Test ---

	//Own multiplexor creating (router)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/status", statusHandler)

	//Timeouts parsing
	readTimeout, err := time.ParseDuration(cfg.ReadTimeout)
	if err != nil {
		log.Fatalf("Wrong format ReadTimeout: %v", err)
	}

	writeTimeout, err := time.ParseDuration(cfg.WriteTimeout)
	if err != nil {
		log.Fatalf("Wrong format WriteTimeout: %v", err)
	}

	idleTimeout, err := time.ParseDuration(cfg.IdleTimeout)
	if err != nil {
		log.Fatalf("Wrong format IdleTimeout: %v", err)
	}

	shutdownTimeout, err := time.ParseDuration(cfg.ShutdownTimeout)
	if err != nil {
		log.Fatalf("Wrong format ShutdownTimeout: %v", err)
	}

	//Create and configure http.Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      mux, //use our own multiplexor
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		ErrorLog:     log.New(os.Stderr, "HTTP-Server Error: ", log.LstdFlags),
	}

	//System sygnals channels and channels for launch error messages
	serverErrors := make(chan error, 1)
	osSignals := make(chan os.Signal, 1)

	//Register signals that we want to listen to (Ctrl+C, SIGTERM)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	//HTTP server launch in separated goroutine
	go func() {
		log.Printf("Web Server launch on %s...\n", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	//Waiting for system signal or server error
	select {
	case err := <-serverErrors:
		//Server returns an error
		if err != http.ErrServerClosed {
			log.Fatalf("HTTP-server launch error: %v", err)
		}
	case sig := <-osSignals:
		//Received system signal
		log.Printf("Received system signal: %v. Begin gracefull shutdown...", sig)

		//Create context with timeout for operation fnish
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		//Initialize gracefull shutdown
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Gracefull shutdown error: %v", err)
			os.Exit(1)
		}
		log.Println("Server shutdown gracefully")
	}

	log.Println("Application closed.")
}

type Config struct {
	ServerPort   int    `yaml:"server_port"`
	BackupDir    string `yaml:"backup_directory"`
	DatabasePath string `yaml:"database_path"`
	SecretSalt   string `yaml:"secret_salt"`

	ReadTimeout  string `yaml:"read_timeout"`
	WriteTimeout string `yaml:"write_timeout"`
	IdleTimeout  string `yaml:"idle_timeout"`

	ShutdownTimeout string `yaml:"shutdown_timeout"`
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Application works correct! Time %s", time.Now().Format("2006-01-02 15:04:05"))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(3 * time.Second)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Overal backup status: Vaiting for main functional realization. (GET /status)")
}
