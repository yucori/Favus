package uploader

import (
	"fmt"
)

func UploadFile(filePath, bucketName, key string) error {
	fmt.Printf("Uploading file %s to bucket %s as %s...\n", filePath, bucketName, key)

	// TODO: split into chunks, upload, track status
	return nil
}