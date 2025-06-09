package backup

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type BackupResult struct {
	JobID    int
	Status   string
	Message  string
	Duration time.Duration
	Time     time.Time
}

func PerformLocalBackup(jobID int, sourcePath, destinationPath string) BackupResult {
	startTime := time.Now()
	result := BackupResult{
		JobID: jobID,
		Time:  startTime,
	}

	log.Printf("Starting backup for job ID %d from '%s' to '%s'", jobID, sourcePath, destinationPath)

	// Перевірка існування джерела
	srcInfo, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		result.Status = "Error"
		result.Message = fmt.Sprintf("Source '%s' not exist: %v", sourcePath, err)
		log.Printf("Backup error for job ID %d: %s", jobID, result.Message)
		result.Duration = time.Since(startTime)
		return result
	}
	if err != nil {
		result.Status = "Error"
		result.Message = fmt.Sprintf("Access to source error '%s': %v", sourcePath, err)
		log.Printf("Backup error for job ID %d: %s", jobID, result.Message)
		result.Duration = time.Since(startTime)
		return result
	}

	if srcInfo.IsDir() {
		err = os.MkdirAll(destinationPath, 0755)
		if err != nil {
			result.Status = "Помилка"
			result.Message = fmt.Sprintf("Can't create destination folder '%s': %v", destinationPath, err)
			log.Printf("Backup error for job ID %d: %s", jobID, result.Message)
			result.Duration = time.Since(startTime)
			return result
		}
	} else {
		destDir := filepath.Dir(destinationPath)
		err = os.MkdirAll(destDir, 0755)
		if err != nil {
			result.Status = "Error"
			result.Message = fmt.Sprintf("Can't create parent directory for destination file '%s': %v", destDir, err)
			log.Printf("Backup error for job ID %d: %s", jobID, result.Message)
			result.Duration = time.Since(startTime)
			return result
		}
	}

	// Копіювання вмісту
	if srcInfo.IsDir() {
		err = copyDirectory(sourcePath, destinationPath)
	} else {
		err = copyFile(sourcePath, destinationPath)
	}

	if err != nil {
		result.Status = "Error"
		result.Message = fmt.Sprintf("Error during backup: %v", err)
		log.Printf("Backup error for job ID %d: %s", jobID, result.Message)
	} else {
		result.Status = "Success"
		result.Message = fmt.Sprintf("Backup successfully completed.")
		log.Printf("Backup for job ID %d completed successfully.", jobID)
	}

	result.Duration = time.Since(startTime)
	return result
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("can't open source file %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("can't create destination file %s: %w", dst, err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("error copy file data: %w", err)
	}
	return out.Close()
}

func copyDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("can't read source directory %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = os.MkdirAll(dstPath, 0755)
			if err != nil {
				return fmt.Errorf("can't create sub directory %s: %w", dstPath, err)
			}
			err = copyDirectory(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
