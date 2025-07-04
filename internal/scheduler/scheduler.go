package scheduler

import (
	"backup-app/internal/backup"
	"backup-app/internal/database"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type SchedulerManager struct {
	Cron    *cron.Cron
	JobRepo *database.JobRepo
}

func NewSchedulerManager(jobRepo *database.JobRepo) *SchedulerManager {
	c := cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger),
	))
	return &SchedulerManager{
		Cron:    c,
		JobRepo: jobRepo,
	}
}

func (sm *SchedulerManager) Start() {
	sm.Cron.Start()
	log.Println("Scheduler started.")
}

func (sm *SchedulerManager) Stop() {
	sm.Cron.Stop()
	log.Println("Scheduler stopped.")
}

/*
func (sm *SchedulerManager) LoadAndScheduleJobs() {
	log.Println("Loading and scheduling jobs...")
	jobs, err := sm.JobRepo.GetAllJobs()
	if err != nil {
		log.Printf("Scheduler: Error loading jobs from DB: %v", err)
		return
	}

	for _, entry := range sm.Cron.Entries() {
		sm.Cron.Remove(entry.ID)
	}
	log.Println("Existing cron entries cleared.")

	for _, job := range jobs {
		if job.IsActive {
			spec := ""
			switch job.Schedule {
			case "daily":
				spec = "0 0 * * *" // Опівночі щодня (00:00)
			case "weekly":
				spec = "0 0 * * 0" // Опівночі щонеділі (00:00)
			case "monthly":
				spec = "0 0 1 * *" // Опівночі першого числа кожного місяця (00:00)
			case "manual":
				// Ручні завдання не плануємо в cron
				log.Printf("Job '%s' (ID: %d) is manual, skipping scheduling.", job.Name, job.ID)
				continue
			default:
				log.Printf("Scheduler: Unknown schedule '%s' for job ID %d. Skipping.", job.Schedule, job.ID)
				continue
			}

			// Плануємо завдання
			jobID := job.ID
			jobName := job.Name
			sourcePath := job.SourcePath
			destinationPath := job.DestinationPath

			_, err := sm.Cron.AddFunc(spec, func() {
				log.Printf("Scheduler: Initiating scheduled backup for job '%s' (ID: %d)", jobName, jobID)
				result := backup.PerformLocalBackup(jobID, sourcePath, destinationPath)

				err := sm.JobRepo.UpdateJobStatusAndLastRun(result.JobID, result.Status, result.Time)
				if err != nil {
					log.Printf("Scheduler: Failed to update job status for ID %d: %v", result.JobID, err)
				} else {
					log.Printf("Scheduler: Job ID %d status updated to '%s' (Duration: %s)", result.JobID, result.Status, result.Duration.String())
				}
			})
			if err != nil {
				log.Printf("Scheduler: Error adding cron job for '%s' (ID: %d): %v", job.Name, job.ID, err)
			} else {
				log.Printf("Scheduler: Job '%s' (ID: %d) scheduled with spec: '%s'", job.Name, job.ID, spec)
			}
		} else {
			log.Printf("Scheduler: Job '%s' (ID: %d) is inactive, skipping scheduling.", job.Name, job.ID)
		}
	}
	log.Println("All active jobs loaded and scheduled.")
}
*/

func (sm *SchedulerManager) LoadAndScheduleJobs() {
	log.Println("Loading and scheduling jobs...")
	jobs, err := sm.JobRepo.GetAllJobs()
	if err != nil {
		log.Printf("Scheduler: Error loading jobs from DB: %v", err)
		return
	}

	// Очищаємо всі існуючі записи cron, щоб уникнути дублювання при перезавантаженні
	// Цей підхід є простим для розробки. Для продакшену можна відстежувати зміни.
	for _, entry := range sm.Cron.Entries() {
		sm.Cron.Remove(entry.ID)
	}
	log.Println("Existing cron entries cleared.")

	for _, job := range jobs {
		if !job.IsActive { // Перевіряємо, чи завдання не активне
			log.Printf("Scheduler: Job '%s' (ID: %d) is inactive, skipping scheduling.", job.Name, job.ID)
			continue
		}

		if job.Schedule == "manual" || job.Schedule == "" { // "manual" або порожній розклад
			log.Printf("Scheduler: Job '%s' (ID: %d) has manual or empty schedule, skipping cron scheduling.", job.Name, job.ID)
			continue
		}

		// Використовуємо значення Schedule безпосередньо як cron-специфікацію
		spec := job.Schedule

		// Валідація cron-специфікації перед додаванням
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		_, err := parser.Parse(spec)
		if err != nil {
			log.Printf("Scheduler: Invalid cron spec '%s' for job '%s' (ID: %d): %v. Skipping scheduling.", spec, job.Name, job.ID, err)
			// Оновлюємо статус завдання, щоб користувач бачив помилку
			err = sm.JobRepo.UpdateJobStatusAndLastRun(job.ID, "Error chedule", time.Now())
			if err != nil {
				log.Printf("Scheduler: Failed to update job status for invalid cron spec (ID %d): %v", job.ID, err)
			}
			continue
		}

		jobID := job.ID
		jobName := job.Name
		sourcePath := job.SourcePath
		destinationPath := job.DestinationPath

		_, err = sm.Cron.AddFunc(spec, func() {
			log.Printf("Scheduler: Initiating scheduled backup for job '%s' (ID: %d)", jobName, jobID)
			result := backup.PerformLocalBackup(jobID, sourcePath, destinationPath)

			err := sm.JobRepo.UpdateJobStatusAndLastRun(result.JobID, result.Status, result.Time)
			if err != nil {
				log.Printf("Scheduler: Failed to update job status for ID %d: %v", result.JobID, err)
			} else {
				log.Printf("Scheduler: Job ID %d status updated to '%s' (Duration: %s)", result.JobID, result.Status, result.Duration.String())
			}
		})
		if err != nil {
			log.Printf("Scheduler: Error adding cron job for '%s' (ID: %d) with spec '%s': %v", job.Name, job.ID, spec, err)
			// Оновлюємо статус завдання, щоб користувач бачив помилку
			err = sm.JobRepo.UpdateJobStatusAndLastRun(job.ID, "Error planing", time.Now())
			if err != nil {
				log.Printf("Scheduler: Failed to update job status for scheduling error (ID %d): %v", job.ID, err)
			}
		} else {
			log.Printf("Scheduler: Job '%s' (ID: %d) scheduled with spec: '%s'", job.Name, job.ID, spec)
		}
	}
	log.Println("All active jobs loaded and scheduled.")
}
