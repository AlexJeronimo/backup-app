package main

import (
	"backup-app/internal/database"
	"backup-app/internal/handlers"
	"backup-app/internal/scheduler"
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

var templates *template.Template

// TODO: add trhrotling (limits) for resource usage by app, RAM, CPU, Network bandwidth for each job separatelly, network limitation for all system.
// TODO: convert bytes to megabytes/gigabytes/... get it from size and apply automatically
// TODO: what abuot large file copy to network, s3, etc?
// TODO: chunking/splitting, resumable uploads, retries, timeouts to avoid breaches on instable networks
// TODO: modify scheduller to human readeble interface. with simple set time, adn chose daily/,amually/or set which days should be included or by which days should backup occurs
// TODO: add dynamic update to job status
// TODO: fix page view (background width not changed but tasks and another info width more wide than background)
// TODO: add next run field (according to scheduller, or "not set" for manual backup)
// TODO: list of backups should be more tableview and more narrow (should fit in windiows size)
// TODO: add data transfer view for exact backup run (how much data will be copied)
// TODO: add backup size in destination directory
// TODO: add support for MySQL, MSSQL, PostgreSQL (add chose db during install process)
// TODO: add some kind of install process with first configuration what type of DB will be used and with initial admin and password config, another configs that should be set during first install/configuration
// TODO: add migration from one type of DB to another
// TODO: in feature think about creating agents, to handle more independent backups (agnets as source and target with configuring source and destination on agents). But all backups managed from main endpoint (aka server)
// TODO: move all yaml/json configs to DB
// TODO: add to support SFTP
// TODO: add to support of TLS (self signed cert (generate and apply) or set certificate if you have one)
// TODO: add to keep passwords in encrypted mode
// TODO: add admin fucntionality (configure server, add users, create and assign roles)
// TODO: add user support (each user has its own backup portal and access to backup jobs according to assigned role, each user can create self backup jobs) (admin has access everywere)
// TODO: add logs output to admin console (create functionality to see logs with live refresh)

func main() {
	//Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Configuration load error: %v", err)
	}

	//Configure log output
	logDir := filepath.Dir(cfg.LogFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Can't create log directory '%s': %v", logDir, err)
	}

	logFile, err := os.OpenFile(cfg.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Can't open log file '%s': %v", cfg.LogFilePath, err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Configs Output

	fmt.Printf("Configuration loaded:\n")
	fmt.Printf(" Server Port: %d\n", cfg.ServerPort)
	fmt.Printf(" Backup Directory: %s\n", cfg.BackupDir)
	fmt.Printf(" Path to DataBase: %s\n", cfg.DatabasePath)
	fmt.Printf(" Secret salt (first 5 symbols): %s...\n", cfg.SecretSalt[:5])
	fmt.Printf(" Server Timeouts: Read=%s, Write=%s, Idle=%s\n", cfg.ReadTimeout, cfg.WriteTimeout, cfg.IdleTimeout)
	fmt.Printf(" Shutdown Timeout: %s\n", cfg.ShutdownTimeout)
	fmt.Printf(" Path to log file: %s\n", cfg.LogFilePath)

	//Initialize DataBase
	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("DataBase initialization error: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("DataBase close connection error: %v", err)
		} else {
			log.Println("DataBAse connection closed.")
		}
	}()
	// ---

	//--- Load HTML-templates ---

	templates, err = template.ParseFiles(filepath.Join("web", "templates", "layout.html"))
	if err != nil {
		log.Fatalf("Error loading HTML-templates: %v", err)
	}
	log.Println("HTML-templates successfully loaded.")

	// ----

	//--- Repos initialization
	userRepo := database.NewUserRepo(db)
	jobRepo := database.NewJobRepo(db)

	// Scheduler initialization
	schedManager := scheduler.NewSchedulerManager(jobRepo)
	schedManager.Start()

	schedManager.LoadAndScheduleJobs()

	//--- HTTP-server configuration ----

	//Own multiplexor creating (router)
	mux := http.NewServeMux()

	//WebHandlers initialization
	webHandlers := handlers.NewWebHandlers(templates, userRepo, jobRepo)

	//sheduler tasks reload
	webHandlers.SetSchedulerReloadFunc(schedManager.LoadAndScheduleJobs)

	// Static files handling
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	//Handlers for web-pages
	mux.HandleFunc("/", webHandlers.HomeHandler)
	mux.HandleFunc("/jobs", webHandlers.JobsHandler)
	mux.HandleFunc("/hello", webHandlers.HelloHTMXHandler)

	mux.HandleFunc("/jobs/new", webHandlers.CreateJobFormHandler)
	mux.HandleFunc("POST /jobs/new", webHandlers.CreateJobHandler)

	mux.HandleFunc("GET /jobs/edit/{id}", webHandlers.EditJobFormHandler)
	mux.HandleFunc("PUT /jobs/edit/{id}", webHandlers.UpdateJobHandler)

	mux.HandleFunc("DELETE /jobs/delete/{id}", webHandlers.DeleteJobHandler)

	mux.HandleFunc("POST /jobs/run/{id}", webHandlers.RunBackupHandler)

	// sysinfo Handlers
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/status", handlers.StatusHandler)

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

		//stop scheduller
		schedManager.Stop()

		//Create context with timeout for operation fnish
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		//Initialize gracefull shutdown
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Graceful shutdown error: %v", err)
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

	LogFilePath string `yaml:"log_file_path"`
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
