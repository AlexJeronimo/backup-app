package database

import (
	"database/sql"
	"fmt"
	"time"
)

type BackupJob struct {
	ID              int          `json:"id" db:"id"`
	Name            string       `json:"name" db:"name"`
	SourcePath      string       `json:"source_path" db:"source_path"`
	DestinationPath string       `json:"destination_path" db:"destination_path"`
	Schedule        string       `json:"schedule" db:"schedule"`
	IsActive        bool         `json:"is_active" db:"is_active"`
	CreatedAt       sql.NullTime `json:"created_at" db:"created_at"`
	UpdatedAt       sql.NullTime `json:"updated_at" db:"updated_at"`
}

type JobRepo struct {
	db *sql.DB
}

func NewJobRepo(db *sql.DB) *JobRepo {
	return &JobRepo{db: db}
}

func (r *JobRepo) CreateJob(name, sourcePath, destinationPath, schedule string, isActive bool) (*BackupJob, error) {
	now := time.Now()
	query := `INSERT INTO backup_jobs (name, source_path, destination_path, schedule, is_active, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?);`
	result, err := r.db.Exec(query, name, sourcePath, destinationPath, schedule, isActive,
		sql.NullTime{Time: now, Valid: true}.Time.Format(time.RFC3339Nano),
		sql.NullTime{Time: now, Valid: true}.Time.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("backup job insert error '%s': %w", name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting ID of new backup task error: %w", err)
	}

	return &BackupJob{
		ID:              int(id),
		Name:            name,
		SourcePath:      sourcePath,
		DestinationPath: destinationPath,
		Schedule:        schedule,
		IsActive:        isActive,
		CreatedAt:       sql.NullTime{Time: now, Valid: true},
		UpdatedAt:       sql.NullTime{Time: now, Valid: true},
	}, nil

}

func (r *JobRepo) GetJobByID(id int) (*BackupJob, error) {
	var job BackupJob
	var createdAtStr, updatedAtStr string
	query := `SELECT id, name, source_path, destination_path, schedule, is_active, created_at, updated_at
			FROM backup_jobs WHERE id = ?;`
	row := r.db.QueryRow(query, id)

	err := row.Scan(&job.ID, &job.Name, &job.SourcePath, &job.DestinationPath, &job.Schedule, &job.IsActive, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("backup task with ID %d not found", id)
		}
		return nil, fmt.Errorf("error getting backup task with ID %d: %w", id, err)
	}

	parseCreatedAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing created_at for task with ID %d: %w", id, err)
	}
	job.CreatedAt = sql.NullTime{Time: parseCreatedAt, Valid: true}

	parseUpdatedAt, err := time.Parse(time.RFC3339Nano, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing updated_at for task with ID %d: %w", id, err)
	}
	job.UpdatedAt = sql.NullTime{Time: parseUpdatedAt, Valid: true}

	return &job, nil
}

func (r *JobRepo) GetJobByName(name string) (*BackupJob, error) {
	var job BackupJob
	var createdAtStr, updatedAtStr string
	query := `SELECT id, name, source_path, destination_path, schedule, is_active, created_at, updated_at
			FROM backup_jobs WHERE name = ?;`
	row := r.db.QueryRow(query, name)

	err := row.Scan(&job.ID, &job.Name, &job.SourcePath, &job.DestinationPath, &job.Schedule, &job.IsActive, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("backup task with '%s' not found", name)
		}
		return nil, fmt.Errorf("error getting backup task '%s': %w", name, err)
	}

	parseCreatedAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing created_at for task '%s': %w", name, err)
	}
	job.CreatedAt = sql.NullTime{Time: parseCreatedAt, Valid: true}

	parseUpdatedAt, err := time.Parse(time.RFC3339Nano, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing updated_at for task '%s': %w", name, err)
	}
	job.UpdatedAt = sql.NullTime{Time: parseUpdatedAt, Valid: true}

	return &job, nil
}

/* func (r *JobRepo) UpdateJob(job *BackupJob) error {
	job.UpdatedAt = sql.NullTime{Time: time.Now(), Valid: true}
	query := `UPDATE backup_jobs SET
				name = ?, source_path = ?, destination_path = ?,
				schedule = ?, is_active = ?, updated_at = ?
				WHERE id = ?;`

	result, err := r.db.Exec(query, job.Name, job.SourcePath, job.DestinationPath, job.Schedule, job.IsActive,
		job.UpdatedAt.Time.Format("2006-01-02 15:04:05.000"), job.ID)
	if err != nil {
		return fmt.Errorf("error updating backup task '%s' (ID %d):%w", job.Name, job.ID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting updated rows quantity: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("backup task with ID %d not found for update", job.ID)
	}

	return nil
} */

func (r *JobRepo) UpdateJob(id int, name, sourcePath, destinationPath, schedule string, isActive bool) (*BackupJob, error) {
	stmt, err := r.db.Prepare(`
		UPDATE backup_jobs
		SET name = ?, source_path = ?, destination_path = ?, schedule = ?,
		is_active = ?, updated_at = ?
		WHERE id = ?;
	`)
	if err != nil {
		return nil, fmt.Errorf("error prepear UPDATE request: %w", err)
	}
	defer stmt.Close()

	updatedAt := time.Now()
	_, err = stmt.Exec(name, sourcePath, destinationPath, schedule, isActive, updatedAt.Format(time.RFC3339Nano), id)
	if err != nil {
		return nil, fmt.Errorf("error executing UPDATE request: %w", err)
	}

	return r.GetJobByID(id)
}

func (r *JobRepo) DeleteJob(id int) error {
	query := `DELETE FROM backup_jobs WHERE id = ?;`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("error deleting backup task with ID %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting deleted rows quantity: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("backup task with ID %d not found for deleting", id)
	}

	return nil
}

func (r *JobRepo) GetAllJobs() ([]BackupJob, error) {
	query := `SELECT id, name, source_path, destination_path, schedule, is_active, created_at, updated_at FROM backup_jobs;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro getting all bakup jobs: %w", err)
	}
	defer rows.Close()

	var jobs []BackupJob
	for rows.Next() {
		var job BackupJob
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&job.ID, &job.Name, &job.SourcePath, &job.DestinationPath,
			&job.Schedule, &job.IsActive, &createdAtStr, &updatedAtStr); err != nil {
			return nil, fmt.Errorf("erro scaning row of backup task: %w", err)
		}

		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing created_at for task: %w", err)
		}
		job.CreatedAt = sql.NullTime{Time: parsedCreatedAt, Valid: true}

		parsedUpdatedAt, err := time.Parse(time.RFC3339Nano, updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing updated_at for task: %w", err)
		}
		job.UpdatedAt = sql.NullTime{Time: parsedUpdatedAt, Valid: true}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error duirng iteration backup tasks rows: %w", err)
	}

	return jobs, nil
}
