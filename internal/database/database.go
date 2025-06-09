package database

import "database/sql"

func IsUniqueConstraintError(err error) bool {
	if err == sql.ErrNoRows {
		return false
	}
	return err != nil && (err.Error() == "UNIQUE constraint failed: users.username" ||
		err.Error() == "UNIQUE constraint failed: backup_jobs.name" ||
		err.Error() == "UNIQUE constraint failed: backup_jobs.source_path, backup_jobs.destination_path" ||
		err.Error() == "UNIQUE constraint failed: backup_jobs.name" ||
		err.Error() == "UNIQUE constraint failed: backup_jobs.source_path, backup_jobs.destination_path" ||
		err.Error() == "UNIQUE constraint failed: users.username")
}
