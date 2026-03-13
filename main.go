package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/chetan-code/gotodo/internal/config"
	"github.com/chetan-code/gotodo/internal/handler"
	"github.com/chetan-code/gotodo/internal/repository"
	"github.com/gorilla/sessions"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

func initDB(dburl string) *sql.DB {
	// Connection string matches the docker-compose environment variables
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		slog.Error("database_intialization_failed", "error", err)
		os.Exit(1)
	}

	//check if connectoin is alive
	err = db.Ping()
	if err != nil {
		slog.Error("database_connection_ping_failed", "error", err)
		os.Exit(1)
	}

	slog.Info("database_intialisation_success", "url", dburl)

	return db
}

func loggerMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		//logging completion of a request
		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"ip", r.RemoteAddr,
			//imp : how long does it take a req to complete
			"duration", time.Since(start).String(),
		)
	})
}

func routing(mux *http.ServeMux, todoHandler *handler.TodoHandler, authHandler *handler.AuthHandler) {
	//url path router to a fuction via default mux
	mux.HandleFunc("/login", todoHandler.LoginHandler)
	mux.HandleFunc("/auth/google", authHandler.BeginAuth)
	mux.HandleFunc("/auth/google/callback", authHandler.AuthCallbackHandler)
	mux.HandleFunc("/logout", authHandler.LogoutHandler)
	//redirect any root to login
	mux.HandleFunc("/", handler.HomeRedirect)

	//we will protect them - only user with valid auth and jwt can access this routes
	mux.HandleFunc("/todos", authHandler.AuthMiddleware(todoHandler.TaskRequestHandler))
	mux.HandleFunc("/todos/clear", authHandler.AuthMiddleware(todoHandler.ClearAllTasks))
	mux.HandleFunc("/todos/toggle", authHandler.AuthMiddleware(todoHandler.ToggleTask))
	mux.HandleFunc("/todos/delete", authHandler.AuthMiddleware(todoHandler.DeleteTask))
	mux.HandleFunc("/todos/edit", authHandler.AuthMiddleware(todoHandler.EditTask))
	mux.HandleFunc("/workers/invite", authHandler.AuthMiddleware(todoHandler.SendNewInvite))
	mux.HandleFunc("/workers/invite/delete", authHandler.AuthMiddleware(todoHandler.DeleteInvite))
	mux.HandleFunc("/workers/respond", authHandler.AuthMiddleware(todoHandler.RespondToNewInvite))
	mux.HandleFunc("/workers/delete", authHandler.AuthMiddleware(todoHandler.RemoveWorker))
	mux.HandleFunc("/workers/sent", authHandler.AuthMiddleware(todoHandler.FetchSentInvites))

}

func startServer(port string, mux http.Handler) {
	err := http.ListenAndServe(port, mux)
	if err != nil {
		slog.Error("server_start_failed", "error", err)
	}
	slog.Info("server_start_success", "port", port)
}

/*
gothic will create temp cookie using key it will store it for sometime
and when user complete login it will compare it to make sure login
process was completed from this app only \
Protection from cross site request forgery
*/
func setupGothic(c *config.Config) {
	//GOTH google setup
	goth.UseProviders(
		google.New(c.GoogleClientID, c.GoogleClientSecret, c.GoogleCallbackUrl, "email", "profile"),
	)

	//gothic jwt setup for cookie (cross site req forgery avoidance)
	keyJWT := c.JwtSecret
	maxAge := 86400 * 30 //30 days
	isProd := c.IsProd() //set to true for https

	store := sessions.NewCookieStore([]byte(keyJWT))
	store.MaxAge(maxAge)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = isProd

	gothic.Store = store
}

func setupSlog() {
	//Json handler that writes to standard out
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug, //log debug and above
		AddSource: true,            //adds file name and line number
	})

	//Intialise new logger and set it as default for the server
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func main() {

	config, err := config.Load()
	if err != nil {
		slog.Error("config_setup_failed", "error", err)
		os.Exit(1)
	}

	//structured logging
	setupSlog()

	db := initDB(config.DbUrl)
	defer db.Close()

	repo, err := repository.NewTodoRepo(db)
	if err != nil {
		slog.Error("repository_creation_failed", "error", err)
		os.Exit(1)
	}
	todoHandler := handler.NewTodoHandler(repo)
	authHandler := handler.NewAuthHandler(config)

	//athentication
	setupGothic(config)

	//routing
	mux := http.NewServeMux()
	routing(mux, todoHandler, authHandler)

	//middleweare
	wrappedMux := loggerMW(mux)

	startServer(config.Port, wrappedMux)
}
