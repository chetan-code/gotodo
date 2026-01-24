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
	repo *repository.TodoRepo
}

func NewTodoHandler(r *repository.TodoRepo) *TodoHandler {
	return &TodoHandler{repo: r}
}

func HomeRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *TodoHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	//show login page
	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("templates/login.html"))
		tmpl.Execute(w, nil)
		return
	}
}

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

func (h *TodoHandler) TodoHandler(w http.ResponseWriter, r *http.Request) {
	email, err := h.GetEmailFromContext(r)
	if err != nil {
		HomeRedirect(w, r)
		return
	}
	//query param if any
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	switch r.Method {
	case http.MethodPost:
		task := r.FormValue("task")
		if task == "" {
			slog.Error("empty_task",
				"method", r.Method,
				"path", r.URL.Path,
				"ip", r.RemoteAddr,
				"email", email)
			return
		}
		h.repo.AddTask(email, task)

		//check if we have htmx request
		if r.Header.Get("HX-Request") == "true" {
			//only update the part and return no need to redirect
			tasks, _ := h.repo.FetchTasks(email, search, status)
			stats, _ := h.repo.GetStats(email)
			data := struct {
				Tasks []models.Task
				Stats repository.TodoStats
			}{
				Tasks: tasks,
				Stats: stats,
			}
			tmpl := template.Must(template.ParseFiles("templates/todos.html"))
			//we yse "task-list" name we used in html {{block}}
			tmpl.ExecuteTemplate(w, "task-list", data)

			//Append the Stats block with the hx-swap-oob attribute
			//find element with "stats-container" id and replace it
			fmt.Fprint(w, `<div id="stats-container" hx-swap-oob="true" style="display: flex; gap: 20px; margin-bottom: 1rem; font-size: 0.9rem;">`)
			tmpl.ExecuteTemplate(w, "stats-container", data)
			fmt.Fprint(w, `</div>`)
			return
		}

		//self redirection
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return

	case http.MethodGet:
		tasks, _ := h.repo.FetchTasks(email, search, status)
		stats, _ := h.repo.GetStats(email)
		//check if we have htmx request - and update the part
		if r.Header.Get("HX-Request") == "true" {
			//only update the part and return no need to redirect
			tmpl := template.Must(template.ParseFiles("templates/todos.html"))
			//we yse "task-list" name we used in html {{block}}
			tmpl.ExecuteTemplate(w, "task-list", struct{ Tasks []models.Task }{Tasks: tasks})
			return
		}
		//render full page
		//data to send to html
		data := struct {
			Email string
			Tasks []models.Task
			Stats repository.TodoStats
		}{
			Email: email,
			Tasks: tasks,
			Stats: stats,
		}

		//laod and render the template :
		tmpl := template.Must(template.ParseFiles("templates/todos.html"))
		tmpl.Execute(w, data)
	}

}

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
	if r.Header.Get("HX-Request") == "true" {
		//only update the part and return no need to redirect
		tasks, _ := h.repo.FetchTasks(email, "", "")
		stats, _ := h.repo.GetStats(email)
		data := struct {
			Tasks []models.Task
			Stats repository.TodoStats
		}{
			Tasks: tasks,
			Stats: stats,
		}
		tmpl := template.Must(template.ParseFiles("templates/todos.html"))
		//we yse "task-list" name we used in html {{block}}
		tmpl.ExecuteTemplate(w, "task-list", data)

		//Append the Stats block with the hx-swap-oob attribute
		//find element with "stats-container" id and replace it
		fmt.Fprint(w, `<div id="stats-container" hx-swap-oob="true" style="display: flex; gap: 20px; margin-bottom: 1rem; font-size: 0.9rem;">`)
		tmpl.ExecuteTemplate(w, "stats-container", data)
		fmt.Fprint(w, `</div>`)
		return
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
	return
}

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
	if r.Header.Get("HX-Request") == "true" {
		stats, _ := h.repo.GetStats(email)
		data := struct {
			Stats repository.TodoStats
		}{
			Stats: stats,
		}
		tmpl := template.Must(template.ParseFiles("templates/todos.html"))
		//Append the Stats block with the hx-swap-oob attribute
		//find element with "stats-container" id and replace it
		fmt.Fprint(w, `<div id="stats-container" hx-swap-oob="true" style="display: flex; gap: 20px; margin-bottom: 1rem; font-size: 0.9rem;">`)
		tmpl.ExecuteTemplate(w, "stats-container", data)
		fmt.Fprint(w, `</div>`)
		return
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
	return
}

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
	if r.Header.Get("HX-Request") == "true" {
		stats, _ := h.repo.GetStats(email)
		data := struct {
			Stats repository.TodoStats
		}{
			Stats: stats,
		}
		// Since we cleared everything, tasks will be empty
		// Re-render the "task-list" block so the user sees "✨ All caught up!"
		tmpl := template.Must(template.ParseFiles("templates/todos.html"))
		tmpl.ExecuteTemplate(w, "task-list", struct{ Tasks []models.Task }{Tasks: nil})

		//Append the Stats block with the hx-swap-oob attribute
		//find element with "stats-container" id and replace it
		fmt.Fprint(w, `<div id="stats-container" hx-swap-oob="true" style="display: flex; gap: 20px; margin-bottom: 1rem; font-size: 0.9rem;">`)
		tmpl.ExecuteTemplate(w, "stats-container", data)
		fmt.Fprint(w, `</div>`)
		return
	}
	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}
