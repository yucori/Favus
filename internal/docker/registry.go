package docker

import "fmt"

func PushImage(imageName, registry string) error {
	fmt.Printf("Pushing image %s to registry %s\n", imageName, registry)
	// TODO: run `docker push ...`
	return nil
}
