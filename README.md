# GoTodo

A high-performance, multi-user Todo application built with **Go (Golang)**. This project demonstrates a production-ready backend architecture using Clean Architecture principles, secure OAuth2 authentication, and hypermedia-driven UI updates.



## Key Technical Features

* **OAuth2 Identity Management:** Integrated with Google OAuth via `goth` for secure user authentication.
* **State-less JWT Sessions:** Implements JSON Web Tokens (JWT) for session persistence, stored in `HttpOnly` and `Secure` cookies to mitigate XSS and session hijacking.
* **Hypermedia-driven UI (HTMX):** Uses **HTMX** to perform partial DOM updates. This reduces server-side rendering overhead and eliminates the need for heavy client-side JavaScript frameworks.
* **Clean Architecture:** Strict separation of concerns between HTTP Handlers, Data Repositories, and Domain Models.
* **Concurrency & Context:** Leverages Go's `context` package to propagate user identity and cancellation signals across middleware and database layers.



## Tech Stack

* **Language:** Go (Golang) 1.25.5+
* **Database:** PostgreSQL (via `pgx/v5`)
* **Frontend:** Go HTML Templates  + HTMX
* **Auth:** Google OAuth2 + JWT (HS256)
* **Styling:** Pico CSS

## System Architecture

The project follows a modular structure to ensure maintainability and testability:

```text
.
├── internal/
│   ├── handler/      # HTTP Logic, Middleware, and Template Rendering
│   ├── models/       # Shared Domain Objects and JWT Claims
│   └── repository/   # Data Access Layer (Postgres implementation)
├── templates/        # HTML Fragments and HTMX Blocks
├── .env.example      # Environment Configuration template
├── main.go           # Entry point & Dependency Injection
└── go.mod            # Dependency Management

```
## Security Implementations

* **Context-Injection Middleware:** A custom AuthMiddleware intercepts requests, validates the JWT, and injects the user's email into the request.Context for downstream use.

* **Data Isolation:** All database queries are scoped to the authenticated user_email. This ensures that a user can never access or modify another user's data, even if they guess a Task ID.

* **Secure Cookie Policy:** Cookies are configured with HttpOnly and SameSite=Lax to protect against common web vulnerabilities.

* **Environment Security:** Sensitive credentials (Secrets, Database URLs) are managed strictly via .env files and never committed to version control.

## Getting Started
### Prerequisites

* Go 1.21 or higher

* A running PostgreSQL instance

* Google Cloud Console OAuth 2.0 Credentials

### Installation

* Clone the repository:
```
git clone [https://github.com/chetan-code/gotodo.git](https://github.com/chetan-code/gotodo.git)

cd gotodo
```    

* Setup Environment Variables: Create a .env file in the root directory based on .env.example:
```
cp .env.example .env
# Edit .env with your local database and Google OAuth credentials
```
* Install Dependencies:
```
go mod tidy
```
* Run the Application:
```
go run main.go
```
* The server will start on http://localhost:8080 (or on your modified port)