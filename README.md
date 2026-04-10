# go-todolist

A web-based todo list app built with Go, Gin, HTMX, and SQLite.

## Prerequisites

- [Go](https://go.dev/dl/) 1.22 or later

## Installing dependencies

After cloning the repo, download the Go module dependencies:

```sh
go mod download
```

This reads `go.mod` and `go.sum` and fetches all required packages. You only need to do this once (or again if dependencies change).

## Running locally

```sh
go build . && ./go-todolist
```

Open http://localhost:8080 in your browser. The SQLite database (`todos.db`) is created automatically in the current directory. Press Ctrl+C to stop the server.

The server handles `SIGINT` (Ctrl+C) and `SIGTERM` gracefully, allowing in-flight requests to complete before shutting down. This also means the server will stop cleanly in Docker containers.

> **Note:** `go run .` also works for quick iteration, but may not forward Ctrl+C to the server reliably. Use `go build` + run the binary directly if you need clean shutdown.

## Running tests

```sh
go test ./...
```

Tests use Go's built-in `testing` package with no external dependencies. Each test creates a temporary SQLite database that is automatically cleaned up.
