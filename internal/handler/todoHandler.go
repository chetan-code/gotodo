package handler

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/chetan-code/gotodo/internal/models"
	"github.com/chetan-code/gotodo/internal/repository"
)

type TodoHandler struct {
	repo  *repository.TodoRepo
	templ *template.Template //html template to send to the client
}

func NewTodoHandler(r *repository.TodoRepo) *TodoHandler {
	parsedTemplates := template.Must(template.ParseFiles("templates/login.html", "templates/todos.html"))
	return &TodoHandler{repo: r, templ: parsedTemplates}
}

func HomeRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Show the login page
func (h *TodoHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	//show login page
	if r.Method == http.MethodGet {
		h.renderHTMX(w, "login.html", nil)
		return
	}
}

// Get email from the request context prepared after authentication
func (h *TodoHandler) GetEmailFromContext(r *http.Request) (email string, err error) {
	//get email form context (context is prepared by auth middleware)
	val := r.Context().Value(emailKey)

	//convert interface{} to string
	email, ok := val.(string)
	if !ok {
		//issue with user email from token
		slog.Error("error_invalid_email_from_jwt")
		return "", fmt.Errorf("Invalid token conversion")
	}

	return email, nil
}

// Check header of request if it has "HX-Request" set to true
func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Render the htmx template using the prvided name
func (h *TodoHandler) renderHTMX(w http.ResponseWriter, templateName string, data interface{}) {
	err := h.templ.ExecuteTemplate(w, templateName, data)
	if err != nil {
		slog.Error("failed_template_render",
			"error", err,
			"template_name", templateName)
		return
	}
}

// Send a invitation to be a worker for me
func (h *TodoHandler) InviteHandler(w http.ResponseWriter, r *http.Request) {
	managerEmail, err := h.GetEmailFromContext(r)
	workerEmail := r.FormValue("worker_email")
	if err != nil {
		HomeRedirect(w, r)
		return
	}
	if workerEmail != "" {
		h.repo.SendInvite(managerEmail, workerEmail)
	}
	// HTMX Response: Just clear the input and maybe show a "Sent!" toast
	// For now, we just return an empty string so the form resets if you use hx-on
	w.Write([]byte("Invite Sent!"))
}

// Accept invitation from a manager/boss to be his worker
func (h *TodoHandler) RespondInviteHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	action := r.URL.Query().Get("action") //accepted or rejected
	id, _ := strconv.Atoi(idStr)

	h.repo.RespondToInvite(id, action)
	// HTMX Response: Remove the request card from the UI
	w.Write([]byte(""))
}

// Handle post and get todos request - post/fetch all todos
func (h *TodoHandler) TodoHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.PostNewTask(w, r)
	case http.MethodGet:
		h.FetchAllTasks(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Add new task in the users todos
func (h *TodoHandler) PostNewTask(w http.ResponseWriter, r *http.Request) {
	email, err := h.GetEmailFromContext(r)
	//query param if any
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	if err != nil {
		HomeRedirect(w, r)
		return
	}
	task := r.FormValue("task")
	workerEmail := r.FormValue("worker_email")
	if workerEmail == "" {
		workerEmail = email
	}
	if task == "" {
		slog.Error("empty_task",
			"method", r.Method,
			"path", r.URL.Path,
			"ip", r.RemoteAddr,
			"email", email)
		return
	}
	h.repo.AddTask(email, task, workerEmail)

	//check if we have htmx request
	if isHTMX(r) {
		//only update the part and return no need to redirect
		tasks, _ := h.repo.FetchTasks(email, search, status)
		stats, _ := h.repo.GetStats(email)
		data := struct {
			Email string
			Tasks []models.Task
			Stats repository.TodoStats
		}{
			Email: email,
			Tasks: tasks,
			Stats: stats,
		}
		h.renderHTMX(w, "task-list", data)
		h.renderHTMX(w, "stats-container", data)
		return
	}

	//self redirection
	http.Redirect(w, r, "/todos", http.StatusSeeOther)
	return
}

// Fetach all tasks from the users todos
func (h *TodoHandler) FetchAllTasks(w http.ResponseWriter, r *http.Request) {
	email, err := h.GetEmailFromContext(r)
	//query param if any
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	if err != nil {
		HomeRedirect(w, r)
		return
	}
	tasks, _ := h.repo.FetchTasks(email, search, status)
	stats, _ := h.repo.GetStats(email)
	assignedTask, _ := h.repo.FetchAssignedToMe(email)
	pendingInvites, _ := h.repo.FetchPendingInvites(email)
	myWorkers, _ := h.repo.FetchMyWorkers(email)
	//render full page
	//data to send to html
	data := struct {
		Email          string
		Tasks          []models.Task
		Stats          repository.TodoStats
		AssignedTasks  []models.Task
		PendingInvites []models.Relationship
		MyWorkers      []string
	}{
		Email:          email,
		Tasks:          tasks,
		Stats:          stats,
		AssignedTasks:  assignedTask,
		PendingInvites: pendingInvites,
		MyWorkers:      myWorkers,
	}
	// Single HTMX check
	if isHTMX(r) {
		h.renderHTMX(w, "task-list", data)
		return
	}
	h.renderHTMX(w, "todos.html", data)
	return
}

// Toggle task status from pending to done and vice versa
func (h *TodoHandler) ToggleHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id") //get email form context (context is prepared by auth middleware)
	email, err := h.GetEmailFromContext(r)
	if err != nil {
		HomeRedirect(w, r)
		return
	}

	if id != "" {
		intid, _ := strconv.Atoi(id)
		err := h.repo.ToggleTask(intid, email)
		if err != nil {
			http.Error(w, "Error toggling task!", http.StatusNotAcceptable)
			return
		}
	} else {
		slog.Error("empty_task_id",
			"method", r.Method,
			"path", r.URL.Path,
			"ip", r.RemoteAddr,
			"email", email,
			"id", id)
	}

	//check if we have htmx request - then just update element avoid redirect
	if isHTMX(r) {
		//only update the part and return no need to redirect
		tasks, _ := h.repo.FetchTasks(email, "", "")
		stats, _ := h.repo.GetStats(email)
		assignedTasks, _ := h.repo.FetchAssignedToMe(email)
		data := struct {
			Email         string
			Tasks         []models.Task
			Stats         repository.TodoStats
			AssignedTasks []models.Task
		}{
			Email:         email,
			Tasks:         tasks,
			Stats:         stats,
			AssignedTasks: assignedTasks,
		}
		h.renderHTMX(w, "task-list", data)
		h.renderHTMX(w, "stats-container", data)
		h.renderHTMX(w, "inbox-list", data)
		return
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
	return
}

// Delete task based on id from users todos
func (h *TodoHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	email, err := h.GetEmailFromContext(r)
	if err != nil {
		HomeRedirect(w, r)
		return
	}
	if id != "" {
		intid, _ := strconv.Atoi(id)
		err := h.repo.DeleteTask(intid, email)
		if err != nil {
			http.Error(w, "Error deleting task with id : "+id, http.StatusNotAcceptable)
			return
		}
	}

	//check if we have htmx request - then just update element avoid redirect
	if isHTMX(r) {
		stats, _ := h.repo.GetStats(email)
		data := struct {
			Stats repository.TodoStats
		}{
			Stats: stats,
		}

		h.renderHTMX(w, "stats-container", data)
		return
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
	return
}

// Clear all tasks from users todos
func (h *TodoHandler) ClearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//get email form context (context is prepared by auth middleware)
	email, err := h.GetEmailFromContext(r)
	if err != nil {
		HomeRedirect(w, r)
		return
	}
	err = h.repo.RemoveAllTask(email)
	if err != nil {
		http.Error(w, "Failed to clear tasks", http.StatusInternalServerError)
		return
	}

	// CHECK FOR HTMX REQUEST
	if isHTMX(r) {
		stats, _ := h.repo.GetStats(email)
		data := struct {
			Stats repository.TodoStats
		}{
			Stats: stats,
		}
		// Since we cleared everything, tasks will be empty
		// Re-render the "task-list" block so the user sees "✨ All caught up!"
		h.renderHTMX(w, "task-list", struct{ Tasks []models.Task }{Tasks: nil})
		h.renderHTMX(w, "stats-container", data)
		return
	}
	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}
