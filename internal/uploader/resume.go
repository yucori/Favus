package uploader

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/yucori/Favus/internal/chunker" // Update with your actual module path

	// config 패키지는 ResumeUploader에서 직접 사용하지 않으므로 임포트 제거 (필요시 다시 추가)
	"github.com/yucori/Favus/pkg/utils" // Update with your actual module path
)

// ResumeUploader allows resuming a multipart upload.
type ResumeUploader struct {
	S3Client *s3.S3
	// Logger 필드 제거: utils 패키지 함수를 직접 호출하므로 더 이상 필요 없음
}

// NewResumeUploader creates a new ResumeUploader.
func NewResumeUploader(s3Client *s3.S3) *ResumeUploader { // logger 인자 제거
	return &ResumeUploader{
		S3Client: s3Client,
	}
}

// ResumeUpload resumes a multipart upload from a saved status.
func (ru *ResumeUploader) ResumeUpload(statusFilePath string) error {
	status, err := LoadStatus(statusFilePath)
	if err != nil {
		utils.Error("Failed to load upload status for resume from %s: %v", statusFilePath, err)
		return fmt.Errorf("failed to load upload status for resume: %w", err)
	}

	utils.Info("Resuming upload for file: %s with UploadID: %s", status.FilePath, status.UploadID)

	// ResumeUploader는 Config 객체에 직접 접근할 수 없으므로,
	// 청크 사이즈는 chunker.DefaultChunkSize를 사용하거나
	// UploadStatus에 저장된 청크 사이즈를 사용해야 합니다.
	// 현재는 DefaultChunkSize를 사용합니다.
	fileChunker, err := chunker.NewFileChunker(status.FilePath, chunker.DefaultChunkSize)
	if err != nil {
		utils.Error("Failed to create file chunker for resume for %s: %v", status.FilePath, err)
		return fmt.Errorf("failed to create file chunker for resume: %w", err)
	}
	chunks := fileChunker.Chunks()

	// Ensure the total parts match
	if len(chunks) != status.TotalParts {
		utils.Error("Mismatch in total parts for %s: expected %d, got %d from status. Aborting resume.", status.FilePath, len(chunks), status.TotalParts)
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
			utils.Info("Part %d already completed, skipping.", ch.Index)
			continue
		}

		reader, err := fileChunker.GetChunkReader(ch)
		if err != nil {
			utils.Error("Failed to get chunk reader for part %d of %s: %v", ch.Index, status.FilePath, err)
			return fmt.Errorf("failed to get chunk reader for part %d: %w", ch.Index, err)
		}

		utils.Info("Uploading part %d (offset %d, size %d) for file %s", ch.Index, ch.Offset, ch.Size, status.FilePath)

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
				utils.Error("Failed to upload part %d for %s: %v", ch.Index, status.FilePath, partErr)
				return partErr
			}
			return nil
		})

		if err != nil {
			utils.Error("Failed to upload part %d for %s after retries: %v", ch.Index, status.FilePath, err)
			return fmt.Errorf("failed to upload part %d after retries: %w", ch.Index, err)
		}

		status.AddCompletedPart(ch.Index, *uploadOutput.ETag)
		if err := status.SaveStatus(statusFilePath); err != nil {
			utils.Error("Failed to save status after completing part %d for %s: %v", ch.Index, status.FilePath, err)
			// Non-fatal, but log it
		}
		utils.Info("Successfully uploaded part %d. ETag: %s", ch.Index, *uploadOutput.ETag)

		completedParts = append(completedParts, &s3.CompletedPart{
			PartNumber: aws.Int64(int64(ch.Index)),
			ETag:       uploadOutput.ETag,
		})
	}

	// Complete the multipart upload
	utils.Info("Completing multipart upload for file: %s", status.FilePath)
	_, err = ru.S3Client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(status.Bucket),
		Key:      aws.String(status.Key),
		UploadId: aws.String(status.UploadID),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		utils.Error("Failed to complete multipart upload for %s: %v", status.FilePath, err)
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	utils.Info("Multipart upload completed successfully for %s", status.FilePath)

	// Clean up status file
	if err := os.Remove(statusFilePath); err != nil {
		utils.Error("Failed to remove status file %s: %v", statusFilePath, err)
	}

	return nil
}
