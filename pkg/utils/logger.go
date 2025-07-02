package utils

import "log"

func Info(msg string) {
	log.Printf("[INFO] %s\n", msg)
}

func Error(msg string) {
	log.Printf("[ERROR] %s\n", msg)
}
