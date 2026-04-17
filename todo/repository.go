package todo

// Filter specifies criteria for querying todos.
type Filter struct {
	ListID   int64     // required — scopes queries to a list
	Status   *bool     // nil=all, true=completed, false=active
	Priority *Priority // nil=all
}

// Repository defines the data access interface for lists and todos.
type Repository interface {
	// List management
	CreateList(name string) (*List, error)
	GetDefaultList() (*List, error)
	GetAllLists() ([]List, error)
	GetList(id int64) (*List, error)
	UpdateList(id int64, name string) (*List, error)
	DeleteList(id int64) error

	// Todo management
	Create(listID int64, title, description string, priority Priority) (*Todo, error)
	List(filter Filter) ([]Todo, error)
	ToggleCompleted(id int64) (*Todo, error)
	Delete(id int64) error
}
