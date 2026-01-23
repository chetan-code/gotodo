package handler

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/chetan-code/webserver/internal/models"
	"github.com/chetan-code/webserver/internal/repository"
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

func (h *TodoHandler) TodoHandler(w http.ResponseWriter, r *http.Request) {
	//get email form context (context is prepared by auth middleware)
	val := r.Context().Value(emailKey)

	//convert interface{} to string
	email, ok := val.(string) //read email from query param
	if !ok {
		//issue with user email from token
		fmt.Println("[This should never happen] : Issue with user email from token")
		HomeRedirect(w, r)
		return
	}

	//handle post request -> add todo to our fake db
	if r.Method == http.MethodPost {
		task := r.FormValue("task")
		if task == "" {
			fmt.Printf("Empty task")
			return
		}
		h.repo.AddTaskDB(email, task)

		//check if we have htmx request
		if r.Header.Get("HX-Request") == "true" {
			//only update the part and return no need to redirect
			tasks, _ := h.repo.FetchTask(email)
			tmpl := template.Must(template.ParseFiles("templates/todos.html"))
			//we yse "task-list" name we used in html {{block}}
			tmpl.ExecuteTemplate(w, "task-list", struct{ Tasks []models.Task }{Tasks: tasks})
			return
		}

		//self redirection
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	//NORMAL GET REQUEST
	//use verified email for db query
	tasks, _ := h.repo.FetchTask(email)

	//data to send to html
	data := struct {
		Email string
		Tasks []models.Task
	}{
		Email: email,
		Tasks: tasks,
	}

	//laod and render the template :
	tmpl := template.Must(template.ParseFiles("templates/todos.html"))
	tmpl.Execute(w, data)
}

func (h *TodoHandler) ToggleHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id") //get email form context (context is prepared by auth middleware)
	val := r.Context().Value(emailKey)

	//convert interface{} to string
	email, ok := val.(string) //read email from query param
	if !ok {
		//issue with user email from token
		fmt.Println("[This should never happen] : Issue with user email from token")
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
	}

	//check if we have htmx request - then just update element avoid redirect
	if r.Header.Get("HX-Request") == "true" {
		//only update the part and return no need to redirect
		tasks, _ := h.repo.FetchTask(email)
		tmpl := template.Must(template.ParseFiles("templates/todos.html"))
		//we yse "task-list" name we used in html {{block}}
		tmpl.ExecuteTemplate(w, "task-list", map[string]interface{}{"Tasks": tasks})
		return
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
	return
}

func (h *TodoHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	//get email form context (context is prepared by auth middleware)
	val := r.Context().Value(emailKey)

	//convert interface{} to string
	email, ok := val.(string) //read email from query param
	if !ok {
		//issue with user email from token
		fmt.Println("[This should never happen] : Issue with user email from token")
		HomeRedirect(w, r)
		return
	}
	if id != "" {
		intid, _ := strconv.Atoi(id)
		err := h.repo.DeleteTask(intid, email)
		if err != nil {
			http.Error(w, "error deleting task! : "+err.Error(), http.StatusNotAcceptable)
			return
		}
	}

	//check if we have htmx request - then just update element avoid redirect
	if r.Header.Get("HX-Request") == "true" {
		//just return success - htmx will disappear it (swap outerHTML)
		w.WriteHeader(http.StatusOK)
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
	val := r.Context().Value(emailKey)
	//convert interface{} to string
	email, ok := val.(string) //read email from query param
	if !ok {
		//issue with user email from token
		fmt.Println("[This should never happen] : Issue with user email from token")
		HomeRedirect(w, r)
		return
	}
	err := h.repo.RemoveAllTask(email)
	if err != nil {
		log.Printf("Error removing task for email : %s \n", email)
		http.Error(w, "Failed to clear tasks", http.StatusInternalServerError)
		return
	}

	// CHECK FOR HTMX REQUEST
	if r.Header.Get("HX-Request") == "true" {
		// Since we cleared everything, tasks will be empty
		// Re-render the "task-list" block so the user sees "✨ All caught up!"
		tmpl := template.Must(template.ParseFiles("templates/todos.html"))
		tmpl.ExecuteTemplate(w, "task-list", struct{ Tasks []models.Task }{Tasks: nil})
		return
	}
	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}
