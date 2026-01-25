package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/chetan-code/gotodo/internal/handler"
	"github.com/chetan-code/gotodo/internal/repository"
	"github.com/gorilla/sessions"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

func loadEnvVar() {
	//load env variables
	err := godotenv.Load()
	if err != nil {
		slog.Error("environment_var_load_failure", "error", err)
		os.Exit(1)
	}
}

func initDB() *sql.DB {
	// Connection string matches the docker-compose environment variables
	dburl := os.Getenv("DB_URL")
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

func routing(mux *http.ServeMux, h *handler.TodoHandler) {
	//url path router to a fuction via default mux
	mux.HandleFunc("/login", h.LoginHandler)
	mux.HandleFunc("/auth/google", handler.BeginAuth)
	mux.HandleFunc("/auth/google/callback", handler.AuthCallbackHandler)
	mux.HandleFunc("/logout", handler.LogoutHandler)
	//redirect any root to login
	mux.HandleFunc("/", handler.HomeRedirect)

	//we will protect them - only user with valid auth and jwt can access this routes
	mux.HandleFunc("/todos", handler.AuthMiddleware(h.TodoHandler))
	mux.HandleFunc("/todos/clear", handler.AuthMiddleware(h.ClearHandler))
	mux.HandleFunc("/todos/toggle", handler.AuthMiddleware(h.ToggleHandler))
	mux.HandleFunc("/todos/delete", handler.AuthMiddleware(h.DeleteHandler))
	mux.HandleFunc("/workers/invite", handler.AuthMiddleware(h.InviteHandler))
	mux.HandleFunc("/workers/respond", handler.AuthMiddleware(h.RespondInviteHandler))

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
func setupGothic() {
	//GOTH google setup
	key := os.Getenv("GOOGLE_CLIENT_ID")
	secret := os.Getenv("GOOGLE_CLIENT_SECRET")
	callback := os.Getenv("GOOGLE_CALLBACK_URL")
	goth.UseProviders(
		google.New(key, secret, callback, "email", "profile"),
	)

	//gothic jwt setup for cookie (cross site req forgery avoidance)
	keyJWT := os.Getenv("JWT_SECRET")
	maxAge := 86400 * 30 //30 days
	isProd := false      //set to true for https

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

	//structure logging
	setupSlog()

	loadEnvVar()

	db := initDB()
	defer db.Close()

	repo, err := repository.NewTodoRepo(db)
	if err != nil {
		slog.Error("repository_creation_failed", "error", err)
		os.Exit(1)
	}
	h := handler.NewTodoHandler(repo)

	//athentication
	setupGothic()

	//routing
	mux := http.NewServeMux()
	routing(mux, h)

	//middleweare
	wrappedMux := loggerMW(mux)

	startServer(":8080", wrappedMux)
}
