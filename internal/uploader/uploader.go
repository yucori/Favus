package uploader

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/your-org/favus/internal/chunker" // Update with your actual module path
	"github.com/your-org/favus/internal/config"   // Update with your actual module path
	"github.com/your-org/favus/pkg/utils"         // Update with your actual module path
)

// S3Uploader handles file uploads and deletions to S3.
type S3Uploader struct {
	S3Client *s3.S3
	Config   *config.Config
	Logger   *utils.Logger
}

// NewS3Uploader creates a new S3Uploader instance.
func NewS3Uploader(cfg *config.Config, logger *utils.Logger) (*S3Uploader, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}
	return &S3Uploader{
		S3Client: s3.New(sess),
		Config:   cfg,
		Logger:   logger,
	}, nil
}

// UploadFile performs a multipart upload of a file to S3.
func (u *S3Uploader) UploadFile(filePath, s3Key string) error {
	u.Logger.Info("Starting multipart upload for file: %s to s3://%s/%s", filePath, u.Config.S3BucketName, s3Key)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("cannot upload empty file: %s", filePath)
	}

	fileChunker, err := chunker.NewFileChunker(filePath, chunker.DefaultChunkSize)
	if err != nil {
		return fmt.Errorf("failed to create file chunker: %w", err)
	}
	chunks := fileChunker.Chunks()

	// 1. Initiate Multipart Upload
	initiateOutput, err := u.S3Client.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket: aws.String(u.Config.S3BucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("failed to initiate multipart upload: %w", err)
	}
	uploadID := *initiateOutput.UploadId
	u.Logger.Info("Initiated multipart upload with UploadID: %s", uploadID)

	// Create a status tracker
	statusFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.upload_status", filepath.Base(filePath)))
	status := NewUploadStatus(filePath, u.Config.S3BucketName, s3Key, uploadID, len(chunks))

	var completedParts []*s3.CompletedPart
	for _, ch := range chunks {
		reader, err := fileChunker.GetChunkReader(ch)
		if err != nil {
			u.Logger.Error("Failed to get chunk reader for part %d: %v", ch.Index, err)
			// Abort upload on critical error
			u.AbortMultipartUpload(s3Key, uploadID)
			return fmt.Errorf("failed to get chunk reader for part %d: %w", ch.Index, err)
		}

		u.Logger.Info("Uploading part %d (offset %d, size %d) for file %s", ch.Index, ch.Offset, ch.Size, filePath)

		var uploadOutput *s3.UploadPartOutput
		err = utils.Retry(5, 2*time.Second, func() error {
			var partErr error
			uploadOutput, partErr = u.S3Client.UploadPart(&s3.UploadPartInput{
				Body:          aws.ReadSeekCloser(reader),
				Bucket:        aws.String(u.Config.S3BucketName),
				Key:           aws.String(s3Key),
				PartNumber:    aws.Int64(int64(ch.Index)),
				UploadId:      aws.String(uploadID),
				ContentLength: aws.Int64(ch.Size),
			})
			if partErr != nil {
				u.Logger.Error("Failed to upload part %d: %v", ch.Index, partErr)
				return partErr
			}
			return nil
		})

		if err != nil {
			u.Logger.Error("Failed to upload part %d after retries: %v", ch.Index, err)
			u.AbortMultipartUpload(s3Key, uploadID)
			return fmt.Errorf("failed to upload part %d after retries: %w", ch.Index, err)
		}

		status.AddCompletedPart(ch.Index, *uploadOutput.ETag)
		if err := status.SaveStatus(statusFilePath); err != nil {
			u.Logger.Error("Failed to save status after completing part %d: %v", ch.Index, err)
			// Non-fatal, but log it
		}

		completedParts = append(completedParts, &s3.CompletedPart{
			PartNumber: aws.Int64(int64(ch.Index)),
			ETag:       uploadOutput.ETag,
		})
		u.Logger.Info("Successfully uploaded part %d. ETag: %s", ch.Index, *uploadOutput.ETag)
	}

	// 3. Complete Multipart Upload
	u.Logger.Info("Completing multipart upload for file: %s", filePath)
	_, err = u.S3Client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(u.Config.S3BucketName),
		Key:      aws.String(s3Key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		u.Logger.Error("Failed to complete multipart upload: %v", err)
		u.AbortMultipartUpload(s3Key, uploadID)
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	u.Logger.Info("Multipart upload completed successfully for %s", filePath)

	// Clean up status file
	if err := os.Remove(statusFilePath); err != nil {
		u.Logger.Error("Failed to remove status file %s: %v", statusFilePath, err)
	}

	return nil
}

// DeleteFile deletes a file from S3.
func (u *S3Uploader) DeleteFile(s3Key string) error {
	u.Logger.Info("Deleting file s3://%s/%s", u.Config.S3BucketName, s3Key)
	_, err := u.S3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(u.Config.S3BucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file %s from S3: %w", s3Key, err)
	}
	u.Logger.Info("Successfully deleted file s3://%s/%s", u.Config.S3BucketName, s3Key)
	return nil
}

// AbortMultipartUpload aborts an ongoing multipart upload.
func (u *S3Uploader) AbortMultipartUpload(s3Key, uploadID string) error {
	u.Logger.Info("Aborting multipart upload for key: %s, UploadID: %s", s3Key, uploadID)
	_, err := u.S3Client.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
		Bucket:   aws.String(u.Config.S3BucketName),
		Key:      aws.String(s3Key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		u.Logger.Error("Failed to abort multipart upload: %v", err)
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}
	u.Logger.Info("Multipart upload aborted successfully for key: %s, UploadID: %s", s3Key, uploadID)
	return nil
}

// ListMultipartUploads lists all ongoing multipart uploads for the bucket.
func (u *S3Uploader) ListMultipartUploads() ([]*s3.MultipartUpload, error) {
	u.Logger.Info("Listing ongoing multipart uploads for bucket: %s", u.Config.S3BucketName)
	output, err := u.S3Client.ListMultipartUploads(&s3.ListMultipartUploadsInput{
		Bucket: aws.String(u.Config.S3BucketName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list multipart uploads: %w", err)
	}
	return output.Uploads, nil
}