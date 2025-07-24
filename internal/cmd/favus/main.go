package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yucori/Favus/internal/config"   // Update with your actual module path
	"github.com/yucori/Favus/internal/uploader" // Update with your actual module path
	"github.com/yucori/Favus/pkg/utils"         // Update with your actual module path
)

func main() {
	logger := utils.NewLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration: %v", err)
	}

	s3Uploader, err := uploader.NewS3Uploader(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize S3 uploader: %v", err)
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: favus <command> [args...]")
		fmt.Println("Commands:")
		fmt.Println("  upload <local_file_path> <s3_key>")
		fmt.Println("  delete <s3_key>")
		fmt.Println("  resume <upload_status_file_path>")
		fmt.Println("  list-uploads")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "upload":
		if len(os.Args) != 4 {
			logger.Fatal("Usage: favus upload <local_file_path> <s3_key>")
		}
		localFilePath := os.Args[2]
		s3Key := os.Args[3]
		if err := s3Uploader.UploadFile(localFilePath, s3Key); err != nil {
			logger.Fatal("Upload failed: %v", err)
		}
		logger.Info("File uploaded successfully.")
	case "delete":
		if len(os.Args) != 3 {
			logger.Fatal("Usage: favus delete <s3_key>")
		}
		s3Key := os.Args[2]
		if err := s3Uploader.DeleteFile(s3Key); err != nil {
			logger.Fatal("Deletion failed: %v", err)
		}
		logger.Info("File deleted successfully.")
	case "resume":
		if len(os.Args) != 3 {
			logger.Fatal("Usage: favus resume <upload_status_file_path>")
		}
		statusFilePath := os.Args[2]
		resumeUploader := uploader.NewResumeUploader(s3Uploader.S3Client, logger)
		if err := resumeUploader.ResumeUpload(statusFilePath); err != nil {
			logger.Fatal("Resume upload failed: %v", err)
		}
		logger.Info("Upload resumed and completed successfully.")
	case "list-uploads":
		if len(os.Args) != 2 {
			logger.Fatal("Usage: favus list-uploads")
		}
		uploads, err := s3Uploader.ListMultipartUploads()
		if err != nil {
			logger.Fatal("Failed to list multipart uploads: %v", err)
		}
		if len(uploads) == 0 {
			logger.Info("No ongoing multipart uploads found.")
			return
		}
		logger.Info("Ongoing multipart uploads:")
		for _, upload := range uploads {
			logger.Info("  UploadID: %s, Key: %s, Initiated: %s", *upload.UploadId, *upload.Key, upload.Initiated.Format(time.RFC3339))
		}
	default:
		logger.Fatal("Unknown command: %s", command)
	}
}
