# GoTodo

A high-performance, multi-user Todo application built with **Go (Golang)**. This project demonstrates a production-ready backend architecture using Clean Architecture principles, secure OAuth2 authentication, and hypermedia-driven UI updates.



## Key Technical Features

* **OAuth2 Identity Management:** Integrated with Google OAuth via `goth` for secure user authentication.
* **State-less JWT Sessions:** Implements JSON Web Tokens (JWT) for session persistence, stored in `HttpOnly` and `Secure` cookies to mitigate XSS and session hijacking.
* **Hypermedia-driven UI (HTMX):** Uses **HTMX** to perform partial DOM updates. This reduces server-side rendering overhead and eliminates the need for heavy client-side JavaScript frameworks.
* **Clean Architecture:** Strict separation of concerns between HTTP Handlers, Data Repositories, and Domain Models.
* **Concurrency & Context:** Leverages Go's `context` package to propagate user identity and cancellation signals across middleware and database layers.



## Tech Stack

* **Language:** Go (Golang) 1.21+
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