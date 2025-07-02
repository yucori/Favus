package chunker

import "fmt"

const DefaultChunkSize = 5 * 1024 * 1024 // 5MB

func SplitFile(path string) ([]string, error) {
	fmt.Printf("Splitting file: %s into %dMB chunks...\n", path, DefaultChunkSize/1024/1024)

	// TODO: Actually read and split file into chunks
	return []string{"chunk1", "chunk2"}, nil
}
