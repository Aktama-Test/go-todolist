package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-todolist/todo"
)

type Handler struct {
	repo      todo.Repository
	tmpl      *template.Template
	defaultID int64
}

func New(repo todo.Repository, tmpl *template.Template, defaultListID int64) *Handler {
	return &Handler{repo: repo, tmpl: tmpl, defaultID: defaultListID}
}

func (h *Handler) Index(c *gin.Context) {
	filter := h.parseFilter(c)
	todos, err := h.repo.List(filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load todos")
		return
	}
	c.HTML(http.StatusOK, "index.html", gin.H{"Todos": todos})
}

func (h *Handler) List(c *gin.Context) {
	filter := h.parseFilter(c)
	todos, err := h.repo.List(filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load todos")
		return
	}
	h.tmpl.ExecuteTemplate(c.Writer, "todo_list", todos)
}

func (h *Handler) Create(c *gin.Context) {
	title := c.PostForm("title")
	description := c.PostForm("description")
	priority := todo.Priority(c.PostForm("priority"))

	if title == "" {
		c.String(http.StatusBadRequest, "Title is required")
		return
	}

	t, err := h.repo.Create(h.defaultID, title, description, priority)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create todo")
		return
	}
	h.tmpl.ExecuteTemplate(c.Writer, "todo_item", t)
}

func (h *Handler) Toggle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid ID")
		return
	}

	t, err := h.repo.ToggleCompleted(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to toggle todo")
		return
	}
	h.tmpl.ExecuteTemplate(c.Writer, "todo_item", t)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := h.repo.Delete(id); err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete todo")
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) parseFilter(c *gin.Context) todo.Filter {
	filter := todo.Filter{ListID: h.defaultID}

	if status := c.Query("filter_status"); status != "" {
		switch status {
		case "completed":
			v := true
			filter.Status = &v
		case "active":
			v := false
			filter.Status = &v
		}
	}

	if priority := c.Query("filter_priority"); priority != "" {
		p := todo.Priority(priority)
		filter.Priority = &p
	}

	return filter
}
