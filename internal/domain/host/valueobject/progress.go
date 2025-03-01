package valueobject

import "encoding/json"

// ProgressCallback is a function that will be called to report progress
type ProgressCallback func(current, total int64)

// Progress represents the current progress of an operation
type Progress struct {
	Current int64  `json:"current"`
	Total   int64  `json:"total"`
	Status  string `json:"status"`
}

// ProgressEvent represents a SSE event for progress updates
type ProgressEvent struct {
	Event string    `json:"event"`
	Data  *Progress `json:"data"`
}

// ToJSON converts the progress event to JSON string
func (pe *ProgressEvent) ToJSON() string {
	data, _ := json.Marshal(pe)
	return string(data)
}

// ProgressTracker is an interface for tracking progress of operations
type ProgressTracker interface {
	// OnProgress is called to report progress
	OnProgress(current, total int64)
}
