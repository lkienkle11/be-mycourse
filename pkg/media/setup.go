package media

import (
	"fmt"

	"mycourse-io-be/pkg/entities"
)

var Cloud *entities.CloudClients

// Setup initializes media SDK clients once at app startup.
func Setup() error {
	client, err := NewCloudClientsFromSetting()
	if err != nil {
		return fmt.Errorf("setup media clients failed: %w", err)
	}
	Cloud = client
	return nil
}
