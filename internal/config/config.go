package config

import (
	"fmt"
	"os"
	"strconv"
)

const DefaultChunkSize = 1024 * 1024 // 1 MB

type Config struct {
	AwsRegion    string
	S3BucketName string
	ChunkSize    int64
}

func LoadConfig() (*Config, error) {
	region := os.Getenv("AWS_REGION")
	bucketName := os.Getenv("S3_BUCKET_NAME")
	chunkSizeStr := os.Getenv("CHUNK_SIZE")

	if region == "" {
		return nil, fmt.Errorf("AWS_REGION environment variable is not set")
	}

	if bucketName == "" {
		return nil, fmt.Errorf("S3_BUCKET_NAME environment variable is not set")
	}

	// Default chunk size
	var chunkSize int64 = DefaultChunkSize
	if chunkSizeStr != "" {
		parsedSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
		if err != nil {
			// 환경 변수가 있지만 숫자로 파싱할 수 없는 경우 경고를 출력하고 기본값 사용
			fmt.Printf("Warning: CHUNK_SIZE environment variable '%s' is not a valid number. Using default chunk size (%d bytes).\n", chunkSizeStr, DefaultChunkSize)
		} else if parsedSize <= 0 {
			// 파싱되었지만 0 이하인 경우 경고를 출력하고 기본값 사용
			fmt.Printf("Warning: CHUNK_SIZE environment variable '%s' must be greater than 0. Using default chunk size (%d bytes).\n", chunkSizeStr, DefaultChunkSize)
		} else {
			chunkSize = parsedSize
		}
	}

	return &Config{
		AwsRegion:    region,
		S3BucketName: bucketName,
		ChunkSize:    chunkSize,
	}, nil
}
