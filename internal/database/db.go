package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("DB directory creation error '%s': %w", dbDir, err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_loc=auto")
	if err != nil {
		return nil, fmt.Errorf("database open error '%s': %w", dbPath, err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("database connection check error: %w", err)
	}

	log.Printf("DataBase SQLite successfully connected: %s", dbPath)

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("DB migrations operation error: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		executed_at TEXT DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("table schema_migrations creation error: %w", err)
	}

	var currentVersion int
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations;")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("getting current schema version error: %w", err)
	}

	log.Printf("Current DB schema version: %d", currentVersion)

	migrations := map[int]string{
		1: `
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT NOT NULL UNIQUE,
				password_hash TEXT NOT NULL,
				created_at TEXT DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now', 'localtime'))
			);
			CREATE INDEX idx_users_username ON users(username);
		`,
		2: `
			CREATE TABLE backup_jobs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				source_path TEXT NOT NULL,
				destination_path TEXT NOT NULL,
				schedule TEXT NOT NULL,
				is_active BOOLEAN NOT NULL DEFAULT 1,
				created_at TEXT DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now', 'localtime')),
				updated_at TEXT DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now', 'localtime'))	
			);
			CREATE INDEX idx_backup_jobs_name ON backup_jobs(name);
		`,
		3: `
			CREATE TABLE backup_runs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				job_id INTEGER NOT NULL,
				start_time TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now', 'localtime')),
				end_time TEXT,
				status TEXT NOT NULL,
				message TEXT,
				FOREIGN KEY (job_id) REFERENCES backup_jobs(id) ON DELETE CASCADE
			);
			CREATE INDEX idx_backup_runs_job_id ON backup_runs(job_id);
			CREATE INDEX idx_backup_runs_status ON backup_runs(status);
		`,
	}

	for version := currentVersion + 1; ; version++ {
		sqlScript, exists := migrations[version]
		if !exists {
			break
		}

		log.Printf("Do migration for version %d...", version)
		_, err := db.Exec(sqlScript)
		if err != nil {
			return fmt.Errorf("version migration process error %d: %w", version, err)
		}

		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (?);", version)
		if err != nil {
			return fmt.Errorf("migration verion write error %d: %w", version, err)
		}

		log.Printf("Migration for version %d compleetd successfully.", version)
	}

	log.Println("All DB migrations completed.")
	return nil
}
