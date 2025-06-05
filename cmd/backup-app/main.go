package main

import (
	"backup-app/internal/backup"
	"backup-app/internal/database"
	"context"
	"fmt"
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

	//--- CRUD for user testing ---
	userRepo := database.NewUserRepo(db)

	log.Println("\n--- Testing CRUD for users ---")

	log.Println("Create user admin...")
	adminUser, err := userRepo.CreateUser("admin", "supersecurepassword")
	if err != nil {
		log.Printf("User 'admin' creation error: %v", err)
	} else {
		log.Printf("User 'admin' successfully created: %v", adminUser)
	}

	log.Println("Create user testuser...")
	testUser, err := userRepo.CreateUser("testuser", "supersecurepassword")
	if err != nil {
		log.Printf("User 'adtestuserin' creation error: %v", err)
	} else {
		log.Printf("User 'testuser' successfully created: %v", testUser)
	}

	log.Println("Creating existing user 'admin', expecting to reveive error...")
	_, err = userRepo.CreateUser("admin", "anotherpassword")
	if err != nil {
		log.Printf("Success: Received expected error during user creation: %v", err)
	} else {
		log.Printf("Error: created existing user, not expected.")
	}

	log.Println("Search for user 'admin'...")
	foundAdmin, err := userRepo.GetUserByUsername("admin")
	if err != nil {
		log.Printf("Search user 'admin' error: %v", err)
	} else {
		log.Printf("Found user 'admin': %+v", foundAdmin)
	}

	log.Println("Search for nonexistent user....")
	_, err = userRepo.GetUserByUsername("nonexistent")
	if err != nil {
		log.Printf("Success: Received expected error during nonexistent user search: %v", err)
	} else {
		log.Println("Error: found nonexistent user, not expected.")
	}

	log.Println("User 'admin' authentication with right password...")
	authAdmin, err := userRepo.AuthenticateUser("admin", "supersecurepassword")
	if err != nil {
		log.Printf("Authentication error for 'admin': %v", err)
	} else {
		log.Printf("User 'admin' authentication successfull: %+v", authAdmin)
	}

	log.Println("User 'admin' authentication with wrong password, expected error...")
	_, err = userRepo.AuthenticateUser("admin", "wrongsupersecurepassword")
	if err != nil {
		log.Printf("Successfull: Received expected authentication error for 'admin' with wrong password: %v", err)
	} else {
		log.Println("Error: User 'admin' authentication with wrong password successful, not expected.")
	}

	if testUser != nil {
		log.Println("Update password for user 'testuser'...")
		err = userRepo.UpdateUser(testUser, "newtestpassword")
		if err != nil {
			log.Printf("Error update 'testuser': %v", err)
		} else {
			log.Printf("Password for 'testuser' successfully updated. New hash: %s...", testUser.PasswordHash[:5])
			_, err = userRepo.AuthenticateUser("testuser", "newtestpassword")
			if err != nil {
				log.Printf("Authentication error for 'testuser' with new password: %v", err)
			} else {
				log.Println("Authentication successfull for 'testuser' with new password.")
			}
		}
	}

	log.Println("Get All users...")
	allUsers, err := userRepo.GetAllUsers()
	if err != nil {
		log.Printf("All user getting error: %v", err)
	} else {
		log.Printf("Found %d users:", len(allUsers))
		for _, u := range allUsers {
			createdAtStr := "N/A"
			if u.CreatedAt.Valid {
				createdAtStr = u.CreatedAt.Time.Format("2006-01-02 15:04:05")
			}
			log.Printf(" - ID: %d, Username: %s, CreatedAt: %s", u.ID, u.Username, createdAtStr)
		}
	}

	if testUser != nil {
		log.Printf("Delet user 'testuser' (ID: %d)...", testUser.ID)
		err = userRepo.DeleteUser(testUser.ID)
		if err != nil {
			log.Printf("User delete error for 'testuser': %v", err)
		} else {
			log.Println("User 'testuser' was deleted successfully.")
		}
	}

	log.Println("Check 'testuser' was deleted...")
	_, err = userRepo.GetUserByUsername("testuser")
	if err != nil && (err.Error() == fmt.Sprintf("user '%s' not found", "testuser")) {
		log.Println("Success: User 'testuser' not found after deleting.")
	} else if err != nil {
		log.Printf("Error during user deletion checking for 'testuser': %v", err)
	} else {
		log.Println("Error: User 'testuser' was not deleted.")
	}

	log.Println("---End of CRUD testing for user---")

	//--- End of CRUD testing for user ---

	//--- Testing CRUD for backup tasks ---

	jobRepo := database.NewJobRepo(db)

	log.Println("\n--- Testing CRUD for backup tasks ---")

	log.Println("Creates backup task 'Daily Documents backup'...")
	job1, err := jobRepo.CreateJob("Daily Document Backup", "C:\\Users\\user\\Documents", "E:\\BackupData\\Documents", "daily", true)
	if err != nil {
		log.Printf("Error creating task 'Daily Documents Backup': %v", err)
	} else {
		createdAtStr := "N/A"
		if job1.CreatedAt.Valid {
			createdAtStr = job1.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
		}
		updatedAtStr := "N/A"
		if job1.UpdatedAt.Valid {
			updatedAtStr = job1.UpdatedAt.Time.Format("2006-01-02 15:04:05.000")
		}

		log.Printf("Backup task 'Daily Documents Backup' successfully created: ID: %d, Name: %s, Schedule: %s, Created: %s, Updated: %s",
			job1.ID, job1.Name, job1.Schedule, createdAtStr, updatedAtStr)
	}

	log.Println("Creates backup task 'Weekly Photos Backup'...")
	job2, err := jobRepo.CreateJob("Weekly Photos Backup", "C:\\Users\\user\\Pictures", "E:\\BackupData\\Photos", "weekly", true)
	if err != nil {
		log.Printf("Error creating task 'Weekly Photos Backup': %v", err)
	} else {
		createdAtStr := "N/A"
		if job2.CreatedAt.Valid {
			createdAtStr = job2.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
		}
		updatedAtStr := "N/A"
		if job2.UpdatedAt.Valid {
			updatedAtStr = job2.UpdatedAt.Time.Format("2006-01-02 15:04:05.000")
		}
		log.Printf("Backup task 'Weekly Photos Backup' successfully created: ID: %d, Name: %s, Schedule: %s, Created: %s, Updated: %s",
			job2.ID, job2.Name, job2.Schedule, createdAtStr, updatedAtStr)
	}

	//Try to create existing task

	log.Println("Trying to create backup task 'Daily Documents backup' which exists")
	_, err = jobRepo.CreateJob("Daily Document Backup", "C:\\Users\\user\\Documents", "E:\\BackupData\\Documents", "daily", true)
	if err != nil {
		log.Printf("Success: got expected error during existing task creating: %v", err)
	} else {
		log.Printf("Error: created existing task, not expected.")
	}

	//Get task by ID
	if job1 != nil {
		log.Printf("Search for task by ID %d...", job1.ID)
		foundJob, err := jobRepo.GetJobByID(job1.ID)
		if err != nil {
			log.Printf("Error search task by ID %d: %v", job1.ID, err)
		} else {
			createdAtStr := "N/A"
			if foundJob.CreatedAt.Valid {
				createdAtStr = foundJob.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			updatedAtStr := "N/A"
			if foundJob.UpdatedAt.Valid {
				updatedAtStr = foundJob.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			log.Printf("Found task by ID %d: Name: %s, Source: %s, IsActive: %t, Created: %s, Updated: %s",
				foundJob.ID, foundJob.Name, foundJob.SourcePath, foundJob.IsActive, createdAtStr, updatedAtStr)
		}
	}

	//Get task by name
	if job1 != nil {
		log.Println("Search for task 'Weekly Photos Backup'...")
		foundJobByName, err := jobRepo.GetJobByName("Weekly Photos Backup")
		if err != nil {
			log.Printf("Error search task 'Weekly Photos Backup': %v", err)
		} else {
			createdAtStr := "N/A"
			if foundJobByName.CreatedAt.Valid {
				createdAtStr = foundJobByName.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			updatedAtStr := "N/A"
			if foundJobByName.UpdatedAt.Valid {
				updatedAtStr = foundJobByName.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			log.Printf("Found task 'Weekly Photos Backup': ID %d, Name: %s, Source: %s, IsActive: %t, Created: %s, Updated: %s",
				foundJobByName.ID, foundJobByName.Name, foundJobByName.SourcePath, foundJobByName.IsActive, createdAtStr, updatedAtStr)
		}
	}

	//Update task
	if job1 != nil {
		log.Printf("Update task 'Daily Documents Backup' (ID %d) - change schedule and do inactive...", job1.ID)
		job1.Schedule = "monthly"
		job1.IsActive = false
		err = jobRepo.UpdateJob(job1)
		if err != nil {
			log.Printf("Error during update task 'Daily Documents Backup': %v", err)
		} else {
			updatedJob, _ := jobRepo.GetJobByID(job1.ID)
			updatedAtStr := "N/A"
			if updatedJob.UpdatedAt.Valid {
				updatedAtStr = updatedJob.UpdatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			log.Printf("Task 'Daily Documents Backup' successfully updated. New Schedule: %s, IsActive: %t, Updated: %s",
				updatedJob.Schedule, updatedJob.IsActive, updatedAtStr)
		}
	}

	//Get all tasks
	log.Println("Get all backup tasks...")
	allJobs, err := jobRepo.GetAllJobs()
	if err != nil {
		log.Printf("Error getting all backup tasks: %v", err)
	} else {
		log.Printf("Found %d backup tasks:", len(allJobs))
		for _, j := range allJobs {
			createdAtStr := "N/A"
			if j.CreatedAt.Valid {
				createdAtStr = j.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			updatedAtStr := "N/A"
			if j.UpdatedAt.Valid {
				updatedAtStr = j.UpdatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			log.Printf(" - ID: %d, Name: %s, Active: %t, Created: %s, Updated: %s",
				j.ID, j.Name, j.IsActive, createdAtStr, updatedAtStr)
		}
	}

	//Delete task
	if job2 != nil {
		log.Printf("Deleting task 'Weekly Photos Backup' (ID: %d)...", job2.ID)
	}
	err = jobRepo.DeleteJob(job2.ID)
	if err != nil {
		log.Printf("Error deleting task 'Weekly Photos Backup': %v", err)
	} else {
		log.Println("Task 'Weekly Photos Backup' successfully deleted.")
	}

	log.Println("Check that 'Weekly Photos Backup' was deleted...")
	_, err = jobRepo.GetJobByName("Weekly Photos Backup")
	if err != nil && (err.Error() == fmt.Sprintf("backup task with '%s' not found", "Weekly Photos Backup")) {
		log.Println("Success: task 'Weekly Photos Backup' not found after deleting")
	} else if err != nil {
		log.Printf("Error during checking deleting 'Weekly Photos Backup': %v", err)
	} else {
		log.Println("Error: task 'Weekly Photos Backup' not deleted.")
	}

	log.Println("--- end of testing CRUD for tasks ---")
	//--- End of CRUD testing for backup tasks ---

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

	//---Extended backup test ----

	log.Println("\n--- Test exetended backup logic ---")

	testSourceDir := filepath.Join("E:", "BackupTest_Source")
	testDestDir := filepath.Join(cfg.BackupDir, "BackupTest_Dest")

	os.RemoveAll(testSourceDir) //delete old if exist
	os.RemoveAll(testDestDir)   //delete old if exist
	os.MkdirAll(filepath.Join(testSourceDir, "sub1"), 0755)
	os.MkdirAll(filepath.Join(testSourceDir, "sub2"), 0755)
	os.WriteFile(filepath.Join(testSourceDir, "file1.txt"), []byte("This is file 1."), 0644)
	os.WriteFile(filepath.Join(testSourceDir, "sub1", "file2.txt"), []byte("This is file 2 in sub1."), 0644)
	os.WriteFile(filepath.Join(testSourceDir, "sub2", "file3.log"), []byte("LOg content."), 0644)

	log.Printf("Test directories prepared: %s", testSourceDir)

	log.Println("\n--- First launch CopyDir (full backup) ---")
	var totalCopiedFiles int
	var totalCopiedBytes int64
	backupStartTime := time.Now()
	err = backup.CopyDir(testSourceDir, testDestDir, &totalCopiedFiles, &totalCopiedBytes)
	if err != nil {
		log.Printf("Error during first backup launch: %v", err)
	} else {
		log.Printf("First backup completed successfully. Copied files: %d, Size: %d bytes", totalCopiedFiles, totalCopiedBytes) //TODO: convert bytes to megabytes/gigabytes/... get it from size and apply automatically
	}
	log.Printf("Time of first backup: %s", time.Since(backupStartTime))

	log.Println("\n--- Second launch CopyDir (expecting file skipping) ----")
	var totalCopiedFiles2 int
	var totalCopiedBytes2 int64
	backupStartTime = time.Now()
	err = backup.CopyDir(testSourceDir, testDestDir, &totalCopiedFiles2, &totalCopiedBytes2)
	if err != nil {
		log.Printf("Error during second backup launch: %v", err)
	} else {
		log.Printf("Second backup completed successfully. Copied files: %d, Sizee: %d bytes", totalCopiedFiles2, totalCopiedBytes2)
		if totalCopiedFiles2 == 0 && totalCopiedBytes2 == 0 {
			log.Println("Success: no one file was copied, as expected (optimization work)")
		} else {
			log.Println("Error: Was copied files during second launch, although were no any changes.")
		}
	}
	log.Printf("Second launch execution time: %s", time.Since(backupStartTime))

	log.Println("\n--- Change file in source folder and third launch of backup ---")
	modifiedFilePath := filepath.Join(testSourceDir, "file1.txt")
	os.WriteFile(modifiedFilePath, []byte("This is file 1, updated content!"), 0644)

	log.Printf("Changed file: %s", modifiedFilePath)
	time.Sleep(1 * time.Second)

	var totalCopiedFiles3 int
	var totalCopiedBytes3 int64
	backupStartTime = time.Now()
	err = backup.CopyDir(testSourceDir, testDestDir, &totalCopiedFiles3, &totalCopiedBytes3)
	if err != nil {
		log.Printf("Error during third backup launch: %v", err)
	} else {
		log.Printf("Third backup successfully completed. Copied files: %d, Size: %d bytes", totalCopiedFiles3, totalCopiedBytes3)
		if totalCopiedFiles3 == 1 {
			log.Println("Success: Copied exact 1 file, as expected (update changed file).")
		} else {
			log.Printf("Error: Copied %d files, expected 1", totalCopiedFiles3)
		}
	}
	log.Printf("Third backup execution time: %s", time.Since(backupStartTime))

	log.Println("\n--- Testing GetDirSize ---")
	sourceSize, err := backup.GetDirSIze(testSourceDir)
	if err != nil {
		log.Printf("Error during getting size of source directory: %v", err)
	} else {
		log.Printf("Zise of source directory '%s' is: %d bytes", testSourceDir, sourceSize)
	}

	destSize, err := backup.GetDirSIze(testDestDir)
	if err != nil {
		log.Printf("Error during getting size of destination directory: %v", err)
	} else {
		log.Printf("Zise of destination directory '%s' is: %d bytes", testDestDir, destSize)
	}

	//--- End of Extended backup test

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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Application works correct! Time %s", time.Now().Format("2006-01-02 15:04:05"))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(3 * time.Second)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Overal backup status: Vaiting for main functional realization. (GET /status)")
}
