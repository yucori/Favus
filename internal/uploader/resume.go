package uploader

import "fmt"

// ResumeUpload resumes an interrupted multipart upload.
func ResumeUpload(uploadID string) error {
	fmt.Printf("Resuming upload with ID: %s\n", uploadID)

	// TODO: Load state, continue upload
	return nil
}
