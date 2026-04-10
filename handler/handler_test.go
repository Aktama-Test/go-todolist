package handler

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"go-todolist/todo"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouterDirect(t *testing.T) (*gin.Engine, *todo.SQLiteRepository) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	repo, err := todo.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	tmpl := template.Must(template.New("").Parse(`
		{{define "index.html"}}
		<html><body><div id="todo-list">{{template "todo_list" .Todos}}</div></body></html>
		{{end}}
		{{define "todo_list"}}
		{{range .}}{{template "todo_item" .}}{{end}}
		{{end}}
		{{define "todo_item"}}
		<div id="todo-{{.ID}}" class="todo-item {{if .Completed}}completed{{end}} priority-{{.Priority}}">
		<span>{{.Title}}</span>
		{{if .Completed}}<button>Undo</button>{{else}}<button>Done</button>{{end}}
		</div>
		{{end}}
	`))

	h := New(repo, tmpl, 1)

	r := gin.New()
	r.SetHTMLTemplate(tmpl)
	r.GET("/", h.Index)
	r.GET("/todos", h.List)
	r.POST("/todos", h.Create)
	r.PATCH("/todos/:id/toggle", h.Toggle)
	r.DELETE("/todos/:id", h.Delete)

	return r, repo
}

func TestIndex_ReturnsOK(t *testing.T) {
	r, _ := setupTestRouterDirect(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "todo-list") {
		t.Error("expected response to contain todo-list div")
	}
}

func TestCreate_ReturnsTodoItem(t *testing.T) {
	r, _ := setupTestRouterDirect(t)

	form := url.Values{"title": {"Test task"}, "description": {"A description"}, "priority": {"high"}}
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Test task") {
		t.Error("expected response to contain task title")
	}
	if !strings.Contains(body, "priority-high") {
		t.Error("expected response to contain priority-high class")
	}
}

func TestCreate_EmptyTitle_ReturnsBadRequest(t *testing.T) {
	r, _ := setupTestRouterDirect(t)

	form := url.Values{"title": {""}, "priority": {"medium"}}
	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestToggle_CompletesAndUndo(t *testing.T) {
	r, repo := setupTestRouterDirect(t)

	repo.Create(1, "Toggle me", "", todo.PriorityMedium)

	// Toggle to completed
	req := httptest.NewRequest(http.MethodPatch, "/todos/1/toggle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "completed") {
		t.Error("expected response to contain 'completed' class")
	}
	if !strings.Contains(w.Body.String(), "Undo") {
		t.Error("expected response to contain 'Undo' button")
	}

	// Toggle back to active
	req = httptest.NewRequest(http.MethodPatch, "/todos/1/toggle", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if strings.Contains(w.Body.String(), "completed") {
		t.Error("expected response to not contain 'completed' class after second toggle")
	}
	if !strings.Contains(w.Body.String(), "Done") {
		t.Error("expected response to contain 'Done' button")
	}
}

func TestToggle_InvalidID_ReturnsBadRequest(t *testing.T) {
	r, _ := setupTestRouterDirect(t)

	req := httptest.NewRequest(http.MethodPatch, "/todos/abc/toggle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDelete_RemovesTodo(t *testing.T) {
	r, repo := setupTestRouterDirect(t)

	repo.Create(1, "Delete me", "", todo.PriorityLow)

	req := httptest.NewRequest(http.MethodDelete, "/todos/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify it's gone
	todos, _ := repo.List(todo.Filter{ListID: 1})
	if len(todos) != 0 {
		t.Errorf("expected 0 todos after delete, got %d", len(todos))
	}
}

func TestDelete_NotFound_ReturnsError(t *testing.T) {
	r, _ := setupTestRouterDirect(t)

	req := httptest.NewRequest(http.MethodDelete, "/todos/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestList_WithFilters(t *testing.T) {
	r, repo := setupTestRouterDirect(t)

	repo.Create(1, "Active low", "", todo.PriorityLow)
	t2, _ := repo.Create(1, "Done high", "", todo.PriorityHigh)
	repo.ToggleCompleted(t2.ID)

	// Filter by active status
	req := httptest.NewRequest(http.MethodGet, "/todos?filter_status=active", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Active low") {
		t.Error("expected active todo in response")
	}
	if strings.Contains(body, "Done high") {
		t.Error("did not expect completed todo in active filter")
	}

	// Filter by priority
	req = httptest.NewRequest(http.MethodGet, "/todos?filter_priority=high", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body = w.Body.String()
	if strings.Contains(body, "Active low") {
		t.Error("did not expect low priority todo in high filter")
	}
	if !strings.Contains(body, "Done high") {
		t.Error("expected high priority todo in response")
	}
}
