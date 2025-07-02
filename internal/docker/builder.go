package docker

import "fmt"

func BuildImage(dockerfilePath, imageName string) error {
	fmt.Printf("Building Docker image: %s from %s\n", imageName, dockerfilePath)
	// TODO: run `docker build ...`
	return nil
}
