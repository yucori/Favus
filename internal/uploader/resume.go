package uploader

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/your-org/favus/internal/chunker" // Update with your actual module path
	"github.com/your-org/favus/pkg/utils"        // Update with your actual module path
)

// ResumeUploader allows resuming a multipart upload.
type ResumeUploader struct {
	S3Client *s3.S3
	Logger   *utils.Logger
}

// NewResumeUploader creates a new ResumeUploader.
func NewResumeUploader(s3Client *s3.S3, logger *utils.Logger) *ResumeUploader {
	return &ResumeUploader{
		S3Client: s3Client,
		Logger:   logger,
	}
}

// ResumeUpload resumes a multipart upload from a saved status.
func (ru *ResumeUploader) ResumeUpload(statusFilePath string) error {
	status, err := LoadStatus(statusFilePath)
	if err != nil {
		return fmt.Errorf("failed to load upload status for resume: %w", err)
	}

	ru.Logger.Info("Resuming upload for file: %s with UploadID: %s", status.FilePath, status.UploadID)

	fileChunker, err := chunker.NewFileChunker(status.FilePath, chunker.DefaultChunkSize)
	if err != nil {
		return fmt.Errorf("failed to create file chunker for resume: %w", err)
	}
	chunks := fileChunker.Chunks()

	// Ensure the total parts match
	if len(chunks) != status.TotalParts {
		return fmt.Errorf("mismatch in total parts: expected %d, got %d from status", len(chunks), status.TotalParts)
	}

	completedParts := make([]*s3.CompletedPart, 0, len(status.CompletedParts))
	for partNum, eTag := range status.CompletedParts {
		completedParts = append(completedParts, &s3.CompletedPart{
			PartNumber: aws.Int64(int64(partNum)),
			ETag:       aws.String(eTag),
		})
	}

	// Upload remaining parts
	for _, ch := range chunks {
		if status.IsPartCompleted(ch.Index) {
			ru.Logger.Info("Part %d already completed, skipping.", ch.Index)
			continue
		}

		reader, err := fileChunker.GetChunkReader(ch)
		if err != nil {
			return fmt.Errorf("failed to get chunk reader for part %d: %w", ch.Index, err)
		}

		ru.Logger.Info("Uploading part %d (offset %d, size %d) for file %s", ch.Index, ch.Offset, ch.Size, status.FilePath)

		var uploadOutput *s3.UploadPartOutput
		err = utils.Retry(5, 2*time.Second, func() error {
			var partErr error
			uploadOutput, partErr = ru.S3Client.UploadPart(&s3.UploadPartInput{
				Body:          aws.ReadSeekCloser(reader),
				Bucket:        aws.String(status.Bucket),
				Key:           aws.String(status.Key),
				PartNumber:    aws.Int64(int64(ch.Index)),
				UploadId:      aws.String(status.UploadID),
				ContentLength: aws.Int64(ch.Size),
			})
			if partErr != nil {
				ru.Logger.Error("Failed to upload part %d: %v", ch.Index, partErr)
				return partErr
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to upload part %d after retries: %w", ch.Index, err)
		}

		status.AddCompletedPart(ch.Index, *uploadOutput.ETag)
		if err := status.SaveStatus(statusFilePath); err != nil {
			ru.Logger.Error("Failed to save status after completing part %d: %v", ch.Index, err)
		}
		ru.Logger.Info("Successfully uploaded part %d. ETag: %s", ch.Index, *uploadOutput.ETag)

		completedParts = append(completedParts, &s3.CompletedPart{
			PartNumber: aws.Int64(int64(ch.Index)),
			ETag:       uploadOutput.ETag,
		})
	}

	// Complete the multipart upload
	ru.Logger.Info("Completing multipart upload for file: %s", status.FilePath)
	_, err = ru.S3Client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(status.Bucket),
		Key:      aws.String(status.Key),
		UploadId: aws.String(status.UploadID),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	ru.Logger.Info("Multipart upload completed successfully for %s", status.FilePath)

	// Clean up status file
	if err := os.Remove(statusFilePath); err != nil {
		ru.Logger.Error("Failed to remove status file %s: %v", statusFilePath, err)
	}

	return nil
}