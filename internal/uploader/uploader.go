package uploader

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/yucori/Favus/internal/chunker"
	"github.com/yucori/Favus/internal/config"
	"github.com/yucori/Favus/pkg/utils" // utils 패키지 임포트 유지
)

// S3Uploader handles file uploads and deletions to S3.
type S3Uploader struct {
	S3Client *s3.S3
	Config   *config.Config
	// Logger 필드 제거: utils 패키지 함수를 직접 호출하므로 더 이상 필요 없음
}

// NewS3Uploader creates a new S3Uploader instance.
func NewS3Uploader(cfg *config.Config) (*S3Uploader, error) { // logger 인자 제거
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AwsRegion),
	})
	if err != nil {
		// utils.Fatal 대신 utils.Error를 사용하여 오류를 반환하고,
		// 호출하는 main 함수에서 Fatal 처리하도록 하는 것이 더 유연합니다.
		utils.Error("Failed to create AWS session: %v", err)
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}
	return &S3Uploader{
		S3Client: s3.New(sess),
		Config:   cfg,
	}, nil
}

// UploadFile performs a multipart upload of a file to S3.
func (u *S3Uploader) UploadFile(filePath, s3Key string) error {
	utils.Info("Starting multipart upload for file: %s to s3://%s/%s", filePath, u.Config.S3BucketName, s3Key)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		utils.Error("Failed to get file info for %s: %v", filePath, err)
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.Size() == 0 {
		utils.Error("Cannot upload empty file: %s", filePath)
		return fmt.Errorf("cannot upload empty file: %s", filePath)
	}

	// config에서 청크 사이즈를 가져옵니다.
	fileChunker, err := chunker.NewFileChunker(filePath, u.Config.ChunkSize)
	if err != nil {
		utils.Error("Failed to create file chunker: %v", err)
		return fmt.Errorf("failed to create file chunker: %w", err)
	}
	chunks := fileChunker.Chunks()

	// 1. Initiate Multipart Upload
	initiateOutput, err := u.S3Client.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket: aws.String(u.Config.S3BucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		utils.Error("Failed to initiate multipart upload for %s: %v", s3Key, err)
		return fmt.Errorf("failed to initiate multipart upload: %w", err)
	}
	uploadID := *initiateOutput.UploadId
	utils.Info("Initiated multipart upload with UploadID: %s", uploadID)

	// Create a status tracker
	statusFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.upload_status", filepath.Base(filePath)))
	status := NewUploadStatus(filePath, u.Config.S3BucketName, s3Key, uploadID, len(chunks))

	var completedParts []*s3.CompletedPart
	for _, ch := range chunks {
		reader, err := fileChunker.GetChunkReader(ch)
		if err != nil {
			utils.Error("Failed to get chunk reader for part %d: %v", ch.Index, err)
			u.AbortMultipartUpload(s3Key, uploadID) // Abort upload on critical error
			return fmt.Errorf("failed to get chunk reader for part %d: %w", ch.Index, err)
		}

		utils.Info("Uploading part %d (offset %d, size %d) for file %s", ch.Index, ch.Offset, ch.Size, filePath)

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
				utils.Error("Failed to upload part %d: %v", ch.Index, partErr)
				return partErr
			}
			return nil
		})

		if err != nil {
			utils.Error("Failed to upload part %d after retries: %v", ch.Index, err)
			u.AbortMultipartUpload(s3Key, uploadID)
			return fmt.Errorf("failed to upload part %d after retries: %w", ch.Index, err)
		}

		status.AddCompletedPart(ch.Index, *uploadOutput.ETag)
		if err := status.SaveStatus(statusFilePath); err != nil {
			utils.Error("Failed to save status after completing part %d: %v", ch.Index, err)
			// Non-fatal, but log it
		}

		completedParts = append(completedParts, &s3.CompletedPart{
			PartNumber: aws.Int64(int64(ch.Index)),
			ETag:       uploadOutput.ETag,
		})
		utils.Info("Successfully uploaded part %d. ETag: %s", ch.Index, *uploadOutput.ETag)
	}

	// 3. Complete Multipart Upload
	utils.Info("Completing multipart upload for file: %s", filePath)
	_, err = u.S3Client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(u.Config.S3BucketName),
		Key:      aws.String(s3Key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		utils.Error("Failed to complete multipart upload: %v", err)
		u.AbortMultipartUpload(s3Key, uploadID)
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	utils.Info("Multipart upload completed successfully for %s", filePath)

	// Clean up status file
	if err := os.Remove(statusFilePath); err != nil {
		utils.Error("Failed to remove status file %s: %v", statusFilePath, err)
	}

	return nil
}

// DeleteFile deletes a file from S3.
func (u *S3Uploader) DeleteFile(s3Key string) error {
	utils.Info("Deleting file s3://%s/%s", u.Config.S3BucketName, s3Key)
	_, err := u.S3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(u.Config.S3BucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		utils.Error("Failed to delete file %s from S3: %v", s3Key, err)
		return fmt.Errorf("failed to delete file %s from S3: %w", s3Key, err)
	}
	utils.Info("Successfully deleted file s3://%s/%s", u.Config.S3BucketName, s3Key)
	return nil
}

// AbortMultipartUpload aborts an ongoing multipart upload.
func (u *S3Uploader) AbortMultipartUpload(s3Key, uploadID string) error {
	utils.Info("Aborting multipart upload for key: %s, UploadID: %s", s3Key, uploadID)
	_, err := u.S3Client.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
		Bucket:   aws.String(u.Config.S3BucketName),
		Key:      aws.String(s3Key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		utils.Error("Failed to abort multipart upload: %v", err)
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}
	utils.Info("Multipart upload aborted successfully for key: %s, UploadID: %s", s3Key, uploadID)
	return nil
}

// ListMultipartUploads lists all ongoing multipart uploads for the bucket.
func (u *S3Uploader) ListMultipartUploads() ([]*s3.MultipartUpload, error) {
	utils.Info("Listing ongoing multipart uploads for bucket: %s", u.Config.S3BucketName)
	output, err := u.S3Client.ListMultipartUploads(&s3.ListMultipartUploadsInput{
		Bucket: aws.String(u.Config.S3BucketName),
	})
	if err != nil {
		utils.Error("Failed to list multipart uploads: %v", err)
		return nil, fmt.Errorf("failed to list multipart uploads: %w", err)
	}
	return output.Uploads, nil
}
