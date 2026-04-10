package main

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"go-todolist/handler"
	"go-todolist/todo"
)

//go:embed templates/* static/*
var content embed.FS

func main() {
	repo, err := todo.NewSQLiteRepository("./todos.db")
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()

	defaultList, err := repo.GetDefaultList()
	if err != nil {
		log.Fatal("failed to get default list:", err)
	}

	tmpl := template.Must(template.ParseFS(content, "templates/*.html", "templates/partials/*.html"))

	r := gin.Default()
	r.SetHTMLTemplate(tmpl)

	staticFS, err := fs.Sub(content, "static")
	if err != nil {
		log.Fatal(err)
	}
	r.StaticFS("/static", http.FS(staticFS))

	h := handler.New(repo, tmpl, defaultList.ID)
	r.GET("/", h.Index)
	r.GET("/todos", h.List)
	r.POST("/todos", h.Create)
	r.PATCH("/todos/:id/toggle", h.Toggle)
	r.DELETE("/todos/:id", h.Delete)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Println("Starting server on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("server forced to shutdown:", err)
	}

	log.Println("Server stopped")
}
