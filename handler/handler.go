package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-todolist/todo"
)

type Handler struct {
	repo todo.Repository
	tmpl *template.Template
}

func New(repo todo.Repository, tmpl *template.Template, defaultListID int64) *Handler {
	return &Handler{repo: repo, tmpl: tmpl}
}

func (h *Handler) Index(c *gin.Context) {
	lists, err := h.repo.GetAllLists()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load lists")
		return
	}

	currentListID := h.getCurrentListID(c, lists)
	filter := h.parseFilter(c, currentListID)
	todos, err := h.repo.List(filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load todos")
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Todos":         todos,
		"Lists":         lists,
		"CurrentListID": currentListID,
	})
}

func (h *Handler) List(c *gin.Context) {
	lists, err := h.repo.GetAllLists()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load lists")
		return
	}

	currentListID := h.getCurrentListID(c, lists)
	filter := h.parseFilter(c, currentListID)
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
	listIDStr := c.PostForm("list_id")

	if title == "" {
		c.String(http.StatusBadRequest, "Title is required")
		return
	}

	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	t, err := h.repo.Create(listID, title, description, priority)
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

func (h *Handler) parseFilter(c *gin.Context, listID int64) todo.Filter {
	filter := todo.Filter{ListID: listID}

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

func (h *Handler) getCurrentListID(c *gin.Context, lists []todo.List) int64 {
	// Try to get from query param
	if listIDStr := c.Query("list_id"); listIDStr != "" {
		if listID, err := strconv.ParseInt(listIDStr, 10, 64); err == nil {
			return listID
		}
	}

	// Default to first list (usually ID 1)
	if len(lists) > 0 {
		return lists[0].ID
	}

	return 1
}

// List management handlers

func (h *Handler) GetLists(c *gin.Context) {
	lists, err := h.repo.GetAllLists()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load lists")
		return
	}

	currentListID := h.getCurrentListID(c, lists)
	h.tmpl.ExecuteTemplate(c.Writer, "list_selector", gin.H{
		"Lists":         lists,
		"CurrentListID": currentListID,
	})
}

func (h *Handler) CreateList(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		c.String(http.StatusBadRequest, "List name is required")
		return
	}

	list, err := h.repo.CreateList(name)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create list")
		return
	}

	// Return the new list item
	h.tmpl.ExecuteTemplate(c.Writer, "list_item", gin.H{
		"List":          list,
		"CurrentListID": list.ID,
	})
}

func (h *Handler) UpdateList(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid ID")
		return
	}

	name := c.PostForm("name")
	if name == "" {
		c.String(http.StatusBadRequest, "List name is required")
		return
	}

	list, err := h.repo.UpdateList(id, name)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to update list")
		return
	}

	h.tmpl.ExecuteTemplate(c.Writer, "list_item", gin.H{
		"List":          list,
		"CurrentListID": id,
	})
}

func (h *Handler) DeleteList(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := h.repo.DeleteList(id); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusOK)
}
