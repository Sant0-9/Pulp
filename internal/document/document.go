package document

import (
	"fmt"
	"time"
)

// Document represents a parsed document
type Document struct {
	Content  string
	Preview  string
	Metadata Metadata
}

// Metadata contains document metadata
type Metadata struct {
	Title         string    `json:"title"`
	SourcePath    string    `json:"source_path"`
	SourceFormat  string    `json:"source_format"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	PageCount     *int      `json:"page_count,omitempty"`
	WordCount     int       `json:"word_count"`
	ConvertedAt   time.Time `json:"converted_at"`
}

// FileSizeHuman returns human-readable file size
func (m Metadata) FileSizeHuman() string {
	bytes := m.FileSizeBytes
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
