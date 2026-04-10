package todo

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestNewSQLiteRepository_CreatesFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer repo.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}
}

func TestGetDefaultList(t *testing.T) {
	repo := newTestRepo(t)

	list, err := repo.GetDefaultList()
	if err != nil {
		t.Fatalf("GetDefaultList: %v", err)
	}
	if list.ID != 1 {
		t.Errorf("expected ID 1, got %d", list.ID)
	}
	if list.Name != "My Todos" {
		t.Errorf("expected name 'My Todos', got %q", list.Name)
	}
}

func TestCreateList(t *testing.T) {
	repo := newTestRepo(t)

	list, err := repo.CreateList("Work")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	if list.Name != "Work" {
		t.Errorf("expected name 'Work', got %q", list.Name)
	}
	if list.ID <= 1 {
		t.Errorf("expected ID > 1, got %d", list.ID)
	}
}

func TestCreate(t *testing.T) {
	repo := newTestRepo(t)

	todo, err := repo.Create(1, "Buy milk", "From the store", PriorityHigh)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if todo.Title != "Buy milk" {
		t.Errorf("expected title 'Buy milk', got %q", todo.Title)
	}
	if todo.Description != "From the store" {
		t.Errorf("expected description 'From the store', got %q", todo.Description)
	}
	if todo.Priority != PriorityHigh {
		t.Errorf("expected priority high, got %q", todo.Priority)
	}
	if todo.Completed {
		t.Error("expected completed to be false")
	}
	if todo.ListID != 1 {
		t.Errorf("expected list_id 1, got %d", todo.ListID)
	}
}

func TestList_ReturnsAll(t *testing.T) {
	repo := newTestRepo(t)

	repo.Create(1, "Task 1", "", PriorityLow)
	repo.Create(1, "Task 2", "", PriorityMedium)
	repo.Create(1, "Task 3", "", PriorityHigh)

	todos, err := repo.List(Filter{ListID: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 3 {
		t.Fatalf("expected 3 todos, got %d", len(todos))
	}
}

func TestList_FilterByStatus(t *testing.T) {
	repo := newTestRepo(t)

	repo.Create(1, "Task 1", "", PriorityLow)
	todo2, _ := repo.Create(1, "Task 2", "", PriorityMedium)
	repo.ToggleCompleted(todo2.ID)

	active := false
	todos, err := repo.List(Filter{ListID: 1, Status: &active})
	if err != nil {
		t.Fatalf("List active: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 active todo, got %d", len(todos))
	}
	if todos[0].Title != "Task 1" {
		t.Errorf("expected 'Task 1', got %q", todos[0].Title)
	}

	completed := true
	todos, err = repo.List(Filter{ListID: 1, Status: &completed})
	if err != nil {
		t.Fatalf("List completed: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 completed todo, got %d", len(todos))
	}
	if todos[0].Title != "Task 2" {
		t.Errorf("expected 'Task 2', got %q", todos[0].Title)
	}
}

func TestList_FilterByPriority(t *testing.T) {
	repo := newTestRepo(t)

	repo.Create(1, "Low task", "", PriorityLow)
	repo.Create(1, "High task", "", PriorityHigh)

	p := PriorityHigh
	todos, err := repo.List(Filter{ListID: 1, Priority: &p})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Title != "High task" {
		t.Errorf("expected 'High task', got %q", todos[0].Title)
	}
}

func TestList_ScopedToList(t *testing.T) {
	repo := newTestRepo(t)

	list2, _ := repo.CreateList("Other")
	repo.Create(1, "Default list task", "", PriorityLow)
	repo.Create(list2.ID, "Other list task", "", PriorityLow)

	todos, err := repo.List(Filter{ListID: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo in default list, got %d", len(todos))
	}
	if todos[0].Title != "Default list task" {
		t.Errorf("expected 'Default list task', got %q", todos[0].Title)
	}
}

func TestToggleCompleted(t *testing.T) {
	repo := newTestRepo(t)

	created, _ := repo.Create(1, "Task", "", PriorityMedium)
	if created.Completed {
		t.Fatal("expected new todo to not be completed")
	}

	toggled, err := repo.ToggleCompleted(created.ID)
	if err != nil {
		t.Fatalf("ToggleCompleted: %v", err)
	}
	if !toggled.Completed {
		t.Error("expected todo to be completed after toggle")
	}

	toggledBack, err := repo.ToggleCompleted(created.ID)
	if err != nil {
		t.Fatalf("ToggleCompleted back: %v", err)
	}
	if toggledBack.Completed {
		t.Error("expected todo to not be completed after second toggle")
	}
}

func TestDelete(t *testing.T) {
	repo := newTestRepo(t)

	created, _ := repo.Create(1, "Task", "", PriorityMedium)

	err := repo.Delete(created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	todos, _ := repo.List(Filter{ListID: 1})
	if len(todos) != 0 {
		t.Errorf("expected 0 todos after delete, got %d", len(todos))
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	err := repo.Delete(999)
	if err == nil {
		t.Error("expected error when deleting non-existent todo")
	}
}
