package docker

import (
	"fmt"
	"time"
)

// ImageMetadata holds metadata about a built Docker image.
type ImageMetadata struct {
	Name        string
	Tag         string
	BuiltAt     time.Time
	SizeMB      float64
	Dockerfile  string
	Description string
}

// NewMetadata creates a new ImageMetadata object.
func NewMetadata(name, tag, dockerfile, description string, sizeBytes int64) *ImageMetadata {
	return &ImageMetadata{
		Name:        name,
		Tag:         tag,
		BuiltAt:     time.Now(),
		SizeMB:      float64(sizeBytes) / (1024 * 1024),
		Dockerfile:  dockerfile,
		Description: description,
	}
}

// PrintMetadata outputs metadata in a human-readable format.
func (m *ImageMetadata) PrintMetadata() {
	fmt.Println("=== Docker Image Metadata ===")
	fmt.Printf("Name       : %s\n", m.Name)
	fmt.Printf("Tag        : %s\n", m.Tag)
	fmt.Printf("Built At   : %s\n", m.BuiltAt.Format(time.RFC3339))
	fmt.Printf("Size       : %.2f MB\n", m.SizeMB)
	fmt.Printf("Dockerfile : %s\n", m.Dockerfile)
	fmt.Printf("Description: %s\n", m.Description)
	fmt.Println("=============================")
}
