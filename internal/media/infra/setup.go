package infra

import (
	"fmt"
	"mycourse-io-be/internal/shared/constants"
)

var Cloud *CloudClients

// Setup initializes media SDK clients once at app startup.
func Setup() error {
	client, err := NewCloudClientsFromSetting()
	if err != nil {
		return fmt.Errorf(constants.MsgSetupMediaClientsFailed, err)
	}
	Cloud = client
	return nil
}
