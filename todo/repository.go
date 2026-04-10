package todo

// Filter specifies criteria for querying todos.
type Filter struct {
	ListID   int64     // required — scopes queries to a list
	Status   *bool     // nil=all, true=completed, false=active
	Priority *Priority // nil=all
}

// Repository defines the data access interface for lists and todos.
type Repository interface {
	CreateList(name string) (*List, error)
	GetDefaultList() (*List, error)

	Create(listID int64, title, description string, priority Priority) (*Todo, error)
	List(filter Filter) ([]Todo, error)
	ToggleCompleted(id int64) (*Todo, error)
	Delete(id int64) error
}
