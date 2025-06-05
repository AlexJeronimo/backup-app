package handlers

import (
	"backup-app/internal/database"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

type WebHandlers struct {
	Templates *template.Template

	UserRepo *database.UserRepo
	JobRepo  *database.JobRepo
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

	if err := wh.Templates.ExecuteTemplate(w, "layout.html", nil); err != nil {
		log.Printf("Error rendering home.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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
