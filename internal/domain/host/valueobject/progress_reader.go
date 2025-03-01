package valueobject

import "io"

// ProgressReader wraps an io.Reader to track read progress
type ProgressReader struct {
	Reader     io.Reader
	Total      int64
	OnProgress func(current, total int64)
	current    int64
}

// Read implements io.Reader
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.current += int64(n)
		if pr.OnProgress != nil {
			pr.OnProgress(pr.current, pr.Total)
		}
	}
	return n, err
}
