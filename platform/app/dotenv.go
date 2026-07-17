package app

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// LoadDotenv loads an optional local .env file without overriding process variables.
func LoadDotenv() error {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
