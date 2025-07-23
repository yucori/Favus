package chunker

import (
	"fmt"
	"io"
	"os"
)

const DefaultChunkSize = 5 * 1024 * 1024 // 5 MB

// Chunk represents a part of a file to be uploaded.
type Chunk struct {
	Index    int    // Part number
	Offset   int64  // Starting offset in the file
	Size     int64  // Size of the chunk
	FilePath string // Path to the original file
}

// FileChunker provides methods to chunk a file.
type FileChunker struct {
	filePath  string
	fileSize  int64
	chunkSize int64
}

// NewFileChunker creates a new FileChunker.
func NewFileChunker(filePath string, chunkSize int64) (*FileChunker, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	return &FileChunker{
		filePath:  filePath,
		fileSize:  fileInfo.Size(),
		chunkSize: chunkSize,
	}, nil
}

// Chunks returns a slice of Chunks for the file.
func (fc *FileChunker) Chunks() []Chunk {
	var chunks []Chunk
	for i := 0; ; i++ {
		offset := int64(i) * fc.chunkSize
		remaining := fc.fileSize - offset
		if remaining <= 0 {
			break
		}

		chunkSize := fc.chunkSize
		if remaining < chunkSize {
			chunkSize = remaining
		}

		chunks = append(chunks, Chunk{
			Index:    i + 1, // S3 part numbers start from 1
			Offset:   offset,
			Size:     chunkSize,
			FilePath: fc.filePath,
		})
	}
	return chunks
}

// GetChunkReader returns an io.Reader for a specific chunk.
func (fc *FileChunker) GetChunkReader(chunk Chunk) (io.Reader, error) {
	file, err := os.Open(fc.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	// Seek to the start of the chunk
	_, err = file.Seek(chunk.Offset, io.SeekStart)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to seek to chunk offset: %w", err)
	}
	// Return a limited reader to read only the chunk's size
	return io.LimitReader(file, chunk.Size), nil
}