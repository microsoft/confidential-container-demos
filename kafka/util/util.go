package util

import (
	"log"
	"os"
)

func GetEnv(envName string) string {
	value, exists := os.LookupEnv(envName)
	if !exists {
		log.Println("Environment variable '" + envName + "' is not set.")
		os.Exit(1)
	}
	return value
}
