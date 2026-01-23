package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/chetan-code/webserver/internal/handler"
	"github.com/chetan-code/webserver/internal/repository"
	"github.com/gorilla/sessions"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

func init() {
	//load env variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading environment varibles")
	}

	//in real world app use env variables
	key := os.Getenv("GOOGLE_CLIENT_ID")
	secret := os.Getenv("GOOGLE_CLIENT_SECRET")
	callback := os.Getenv("GOOGLE_CALLBACK_URL")
	goth.UseProviders(
		google.New(key, secret, callback, "email", "profile"),
	)
}

func initDB(dburl string) *sql.DB {
	var err error
	// Connection string matches the docker-compose environment variables
	connStr := dburl
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}

	//check if connectin is alive
	err = db.Ping()
	if err != nil {
		log.Fatal("Can not connect to db : ", err)
	}

	fmt.Println("Connected to PostgresSQL!")

	return db
}

func loggerMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s] %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
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

}

func startServer(port string, mux http.Handler) {
	//start server
	fmt.Println("Server starting on", port)
	err := http.ListenAndServe(port, mux)
	if err != nil {
		fmt.Printf("Error starting server : %s \n", err)
	}
}

/*
gothic will create temp cookie using key it will store it for sometime
and when user complete login it will compare it to make sure login
process was completed from this app only \
Protection from cross site request forgery
*/
func setupGothic() {
	key := os.Getenv("JWT_SECRET")
	maxAge := 86400 * 30 //30 days
	isProd := false      //set to true for https

	store := sessions.NewCookieStore([]byte(key))
	store.MaxAge(maxAge)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = isProd

	gothic.Store = store
}

func main() {
	dburl := os.Getenv("DB_URL")
	db := initDB(dburl)
	defer db.Close()

	repo, err := repository.NewTodoRepo(db)
	if err != nil {
		log.Fatal("Cant create repo :", err)
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
