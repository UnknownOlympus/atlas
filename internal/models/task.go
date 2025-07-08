package models

// Task represents a geocoding task with an ID and an associated address.
type Task struct {
	ID      int    // ID is the unique identifier for the task.
	Address string // Address is the location to be geocoded.
}
