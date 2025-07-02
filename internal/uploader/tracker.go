package uploader

import "fmt"

// TrackProgress is a stub for upload progress tracking.
func TrackProgress(completed, total int64) {
	percentage := float64(completed) / float64(total) * 100
	fmt.Printf("Upload progress: %.2f%%\n", percentage)
}
