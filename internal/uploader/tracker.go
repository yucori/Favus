package uploader

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// UploadStatus represents the status of a multipart upload.
type UploadStatus struct {
	FilePath       string `json:"filePath"`
	UploadID       string `json:"uploadId"`
	Bucket         string `json:"bucket"`
	Key            string `json:"key"`
	CompletedParts map[int]string `json:"completedParts"` // Map of part number to ETag
	TotalParts     int `json:"totalParts"`
	Mu             sync.Mutex `json:"-"` // Mutex to protect concurrent access
}

// NewUploadStatus creates a new UploadStatus.
func NewUploadStatus(filePath, bucket, key, uploadID string, totalParts int) *UploadStatus {
	return &UploadStatus{
		FilePath:       filePath,
		UploadID:       uploadID,
		Bucket:         bucket,
		Key:            key,
		CompletedParts: make(map[int]string),
		TotalParts:     totalParts,
	}
}

// AddCompletedPart adds a completed part to the status.
func (us *UploadStatus) AddCompletedPart(partNumber int, eTag string) {
	us.Mu.Lock()
	defer us.Mu.Unlock()
	us.CompletedParts[partNumber] = eTag
}

// IsPartCompleted checks if a part has been completed.
func (us *UploadStatus) IsPartCompleted(partNumber int) bool {
	us.Mu.Lock()
	defer us.Mu.Unlock()
	_, exists := us.CompletedParts[partNumber]
	return exists
}

// SaveStatus saves the current upload status to a file.
func (us *UploadStatus) SaveStatus(statusFilePath string) error {
	us.Mu.Lock()
	defer us.Mu.Unlock()

	data, err := json.MarshalIndent(us, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal upload status: %w", err)
	}
	return os.WriteFile(statusFilePath, data, 0644)
}

// LoadStatus loads an upload status from a file.
func LoadStatus(statusFilePath string) (*UploadStatus, error) {
	data, err := os.ReadFile(statusFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload status file: %w", err)
	}
	var us UploadStatus
	if err := json.Unmarshal(data, &us); err != nil {
		return nil, fmt.Errorf("failed to unmarshal upload status: %w", err)
	}
	us.Mu = sync.Mutex{} // Initialize mutex after unmarshaling
	return &us, nil
}