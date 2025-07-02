package main

import (
	"fmt"
	"log"

	"favus/internal/uploader"
)

func main() {
	fmt.Println("Favus - S3 Multipart Upload Automation Tool")

	// TODO: parse CLI args or config
	err := uploader.UploadFile("sample.mp4", "my-bucket", "uploads/sample.mp4")
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}
}