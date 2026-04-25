package media

import (
	"fmt"
)

var Cloud *CloudClients

// Setup initializes media SDK clients once at app startup.
func Setup() error {
	client, err := NewCloudClientsFromEnv()
	if err != nil {
		return fmt.Errorf("setup media clients failed: %w", err)
	}
	Cloud = client
	return nil
}
