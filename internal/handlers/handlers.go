package handlers

import (
	"backup-app/internal/backup"
	"backup-app/internal/database"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

type WebHandlers struct {
	Templates *template.Template
	UserRepo  *database.UserRepo
	JobRepo   *database.JobRepo
}

func NewWebHandlers(tmpl *template.Template, userRepo *database.UserRepo, jobRepo *database.JobRepo) *WebHandlers {
	return &WebHandlers{
		Templates: tmpl,
		UserRepo:  userRepo,
		JobRepo:   jobRepo,
	}
}

func (wh *WebHandlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := wh.Templates.Clone()
	if err != nil {
		log.Printf("Template clone error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err = tmpl.ParseFiles(filepath.Join("web", "templates", "home.html"))
	if err != nil {
		log.Printf("Error parsing home.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	jobs, err := wh.JobRepo.GetAllJobs()
	if err != nil {
		log.Printf("Error getting backup tasks: %v", err)
		http.Error(w, "Problem with getting backup tasks", http.StatusInternalServerError)
		return
	}

	data := struct {
		Jobs []database.BackupJob
	}{
		Jobs: jobs,
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error rendering home.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (wh *WebHandlers) CreateJobFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl, err := wh.Templates.Clone()
	if err != nil {
		log.Printf("Template clone error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err = tmpl.ParseFiles(filepath.Join("web", "templates", "create_job.html"))
	if err != nil {
		log.Printf("Error parsing create_job.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", nil); err != nil {
		log.Printf("Error rendering create_job.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (wh *WebHandlers) CreateJobHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method ot allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error form parsing: %v", err)
		http.Error(w, "Form parsing error", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	sourcePath := r.FormValue("source_path")
	destinationPath := r.FormValue("destination_path")
	schedule := r.FormValue("schedule")
	isActiveStr := r.FormValue("is_active")

	isActive := (isActiveStr == "true")

	if name == "" || sourcePath == "" || destinationPath == "" || schedule == "" {
		log.Println("Not all fields filled with necessary info.")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="message error">Error: please fill all necessary fields.</div>`)
		return
	}

	_, err := wh.JobRepo.CreateJob(name, sourcePath, destinationPath, schedule, isActive)
	if err != nil {
		log.Printf("Error creating backup task in DB: %v", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		if database.IsUniqueConstraintError(err) {
			fmt.Fprintf(w, `<div class="message error">Error: Task with this name or paths is exists.</div>`)
		} else {
			fmt.Fprintf(w, `<div class="message error"> Error creating task: %v</div>`, err)
		}
		return
	}

	log.Printf("Successfully created new backup task: %s", name)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
		<div class="message success"> Task "%s" successfully created!</div>
		`, name)
}

func (wh *WebHandlers) JobsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "<h1>Backup Tasks (under construction)</h1><p> Here will be list of your backup tasks.</p>")
}

func (wh *WebHandlers) HelloHTMXHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(500 * time.Millisecond)
	fmt.Fprintf(w, "<p>Hello from Go server! Time: %s</p>", time.Now().Format("15:04:05"))
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Application works correct! Time %s", time.Now().Format("2006-01-02 15:04:05"))
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(3 * time.Second)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Overal backup status: Vaiting for main functional realization. (GET /status)")
}

func (wh *WebHandlers) EditJobFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	jobID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("EditJobFormHandler: Invalid job ID in URL: %v", err)
		http.Error(w, "Incorrect ID request", http.StatusBadRequest)
		return
	}

	job, err := wh.JobRepo.GetJobByID(jobID)
	if err != nil {
		log.Printf("EditJobFormHandler: Error getting job bu ID %d: %v", jobID, err)
		if err.Error() == fmt.Sprintf("backup task with ID %d not found", jobID) {
			http.NotFound(w, r)
		} else {
			http.Error(w, " Can't load task for editing", http.StatusInternalServerError)
		}
		return
	}

	tmpl, err := wh.Templates.Clone()
	if err != nil {
		log.Printf("EditJobFromHandler: Error template cloning: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err = tmpl.ParseFiles(filepath.Join("web", "templates", "edit_job.html"))
	if err != nil {
		log.Printf("EditJobFormHandler: Error parsing edit_job.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Job *database.BackupJob
	}{
		Job: job,
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("EditJobFormHandler: Error rendering edit_job.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	log.Printf("EditJobFormHandler: Successfully rendered edit form for job ID %d.", jobID)
}

func (wh *WebHandlers) UpdateJobHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("UpdateJobHandler: Received PUT request.")
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	jobID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("UpdateJobHandler: Invalid job ID in URL: %v", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="message error">Error: Invalid task ID.</div>`)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("UpdateJobHandler: Error form parsing: %v", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="message error">Error: Can't parse form data.</div>`)
		return
	}

	name := r.FormValue("name")
	sourcePath := r.FormValue("source_path")
	destinationPath := r.FormValue("destination_path")
	schedule := r.FormValue("schedule")
	isActiveStr := r.FormValue("is_active")

	isActive := (isActiveStr == "true")

	log.Printf("UpdateJobHandler: Job ID %d, Form values - Name: %s, Source: %s, Dest: %s, Schedule: %s, Active: %t",
		jobID, name, sourcePath, destinationPath, schedule, isActive)

	if name == "" || sourcePath == "" || destinationPath == "" || schedule == "" {
		log.Println("UpdateJobHandler: Not all fields filled with necessary info. Sending Bad Request.")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="message error">Error: please fill all necessary fields.</div>`)
		return
	}

	_, err = wh.JobRepo.UpdateJob(jobID, name, sourcePath, destinationPath, schedule, isActive)
	if err != nil {
		log.Printf("UpdateJobHandler: Error updating backup task in DB (ID %d): %v", jobID, err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)

		if database.IsUniqueConstraintError(err) {
			fmt.Fprintf(w, `<div class="message error">Error: Task with this name or path exists.</div>`)
		} else {
			fmt.Fprintf(w, `<div class="message error">Error update task: %v</div>`, err)
		}
		return
	}

	log.Printf("UpdateJobHandler: Successfully updated backup task: %s (ID: %d)", name, jobID)
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<div class="message success">Task "%s" successfully updated!</div>`, name)
}

func (wh *WebHandlers) DeleteJobHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("DeleteJobHandler: Received DELETE request.")
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	jobID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("DeleteJobHandler: Invalid job ID in URL: %v", err)
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	err = wh.JobRepo.DeleteJob(jobID)
	if err != nil {
		log.Printf("DeleteJobHandler: Error deleting backup task (ID %d): %v", jobID, err)
		if err.Error() == fmt.Sprintf("task with ID %d not found for deleting", jobID) {
			http.Error(w, fmt.Sprintf("task with ID %d not found.", jobID), http.StatusNotFound)
		} else {
			http.Error(w, "Error deleting task.", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("DeleteJobHandler: Successfully deleted backup task with ID: %d", jobID)
	w.WriteHeader(http.StatusOK)
}

func (wh *WebHandlers) RunBackupHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("RunBackupHandler: Received POST request.")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	jobID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("RunBackupHandler: Invalid job ID in URL: %v", err)
		http.Error(w, "Incorrect ID task", http.StatusBadRequest)
		return
	}

	job, err := wh.JobRepo.GetJobByID(jobID)
	if err != nil {
		log.Printf("RunBackupHandler: Error getting job by ID %d: %v", jobID, err)
		if err.Error() == fmt.Sprintf("task with ID %d not found", jobID) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Can't load task for backup", http.StatusInternalServerError)
		}
		return
	}

	go func() {
		log.Printf("Starting asynchronous backup for job ID %d: %s", job.ID, job.Name)
		result := backup.PerformLocalBackup(job.ID, job.SourcePath, job.DestinationPath)

		err := wh.JobRepo.UpdateJobStatusAndLastRun(result.JobID, result.Status, result.Time)
		if err != nil {
			log.Printf("Failed to update job status for ID %d: %v", result.JobID, err)
		} else {
			log.Printf("Job ID %d status updated to '%s' (Duration: %s)", result.JobID, result.Status, result.Duration.String())
		}
	}()

	log.Printf("RunBackupHandler: Backup initiated for job ID %d. Sending success response.", jobID)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<div class="status-indicator" id="job-status-%d">
                       <span class="status-pending">Backup started...</span>
                     </div>`, jobID)
}
