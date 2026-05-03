package models

import (
	"time"

	"mycourse-io-be/dbschema"
)

// SystemAppConfig is the singleton row (id must be 1) holding isolated system secrets.
// Not related to process .env keys of the same conceptual name.
type SystemAppConfig struct {
	ID                   int       `gorm:"column:id;primaryKey" json:"id"`
	AppCLISystemPassword string    `gorm:"column:app_cli_system_password;not null" json:"-"`
	AppSystemEnv         string    `gorm:"column:app_system_env;not null" json:"-"`
	AppTokenEnv          string    `gorm:"column:app_token_env;not null" json:"-"`
	UpdatedAt            time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SystemAppConfig) TableName() string { return dbschema.System.AppConfig() }
