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
	// logger := utils.NewLogger() // NewLogger가 더 이상 필요 없으므로 제거

	cfg, err := config.LoadConfig()
	if err != nil {
		utils.Fatal("Failed to load configuration: %v", err) // logger.Fatal 대신 utils.Fatal 사용
	}

	s3Uploader, err := uploader.NewS3Uploader(cfg) // logger 인자 제거
	if err != nil {
		utils.Fatal("Failed to initialize S3 uploader: %v", err) // logger.Fatal 대신 utils.Fatal 사용
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
			utils.Fatal("Usage: favus upload <local_file_path> <s3_key>") // logger.Fatal 대신 utils.Fatal 사용
		}
		localFilePath := os.Args[2]
		s3Key := os.Args[3]
		if err := s3Uploader.UploadFile(localFilePath, s3Key); err != nil {
			utils.Fatal("Upload failed: %v", err) // logger.Fatal 대신 utils.Fatal 사용
		}
		utils.Info("File uploaded successfully.") // logger.Info 대신 utils.Info 사용
	case "delete":
		if len(os.Args) != 3 {
			utils.Fatal("Usage: favus delete <s3_key>") // logger.Fatal 대신 utils.Fatal 사용
		}
		s3Key := os.Args[2]
		if err := s3Uploader.DeleteFile(s3Key); err != nil {
			utils.Fatal("Deletion failed: %v", err) // logger.Fatal 대신 utils.Fatal 사용
		}
		utils.Info("File deleted successfully.") // logger.Info 대신 utils.Info 사용
	case "resume":
		if len(os.Args) != 3 {
			utils.Fatal("Usage: favus resume <upload_status_file_path>") // logger.Fatal 대신 utils.Fatal 사용
		}
		statusFilePath := os.Args[2]
		resumeUploader := uploader.NewResumeUploader(s3Uploader.S3Client) // logger 인자 제거
		if err := resumeUploader.ResumeUpload(statusFilePath); err != nil {
			utils.Fatal("Resume upload failed: %v", err) // logger.Fatal 대신 utils.Fatal 사용
		}
		utils.Info("Upload resumed and completed successfully.") // logger.Info 대신 utils.Info 사용
	case "list-uploads":
		if len(os.Args) != 2 {
			utils.Fatal("Usage: favus list-uploads") // logger.Fatal 대신 utils.Fatal 사용
		}
		uploads, err := s3Uploader.ListMultipartUploads()
		if err != nil {
			utils.Fatal("Failed to list multipart uploads: %v", err) // logger.Fatal 대신 utils.Fatal 사용
		}
		if len(uploads) == 0 {
			utils.Info("No ongoing multipart uploads found.") // logger.Info 대신 utils.Info 사용
			return
		}
		utils.Info("Ongoing multipart uploads:") // logger.Info 대신 utils.Info 사용
		for _, upload := range uploads {
			utils.Info("  UploadID: %s, Key: %s, Initiated: %s", *upload.UploadId, *upload.Key, upload.Initiated.Format(time.RFC3339))
		}
	default:
		utils.Fatal("Unknown command: %s", command) // logger.Fatal 대신 utils.Fatal 사용
	}
}
