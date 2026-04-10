package todo

import "time"

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type List struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

type Todo struct {
	ID          int64
	ListID      int64
	Title       string
	Description string
	Completed   bool
	Priority    Priority
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
