package config

import "fmt"

type Config struct {
	Bucket    string
	ChunkSize int64
}

func LoadConfig(path string) (*Config, error) {
	fmt.Printf("Loading config from: %s\n", path)
	// TODO: read YAML or JSON config
	return &Config{
		Bucket:    "my-bucket",
		ChunkSize: 5 * 1024 * 1024,
	}, nil
}
