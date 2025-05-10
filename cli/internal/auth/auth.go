package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var apiKeyFileName = ".cloud-assist-api-key"

// SaveAPIKey saves the API key to a file in the user's home directory
func SaveAPIKey(apiKey string) error {
	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	keyPath := filepath.Join(homeDir, apiKeyFileName)
	return os.WriteFile(keyPath, []byte(apiKey), 0600) // Read/write permissions for user only
}

// GetAPIKey retrieves the API key from the user's home directory
func GetAPIKey() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	keyPath := filepath.Join(homeDir, apiKeyFileName)
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read API key: %w", err)
	}

	apiKey := string(data)
	if apiKey == "" {
		return "", errors.New("stored API key is empty")
	}

	return apiKey, nil
}

// ClearAPIKey removes the stored API key
func ClearAPIKey() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	keyPath := filepath.Join(homeDir, apiKeyFileName)
	_, err = os.Stat(keyPath)
	if os.IsNotExist(err) {
		// File doesn't exist, nothing to do
		return nil
	}

	return os.Remove(keyPath)
}
