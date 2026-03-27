package models

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"mycourse-io-be/pkg/setting"
)

var DB *gorm.DB

func Setup() error {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		setting.DatabaseSetting.Host,
		setting.DatabaseSetting.Port,
		setting.DatabaseSetting.User,
		setting.DatabaseSetting.Password,
		setting.DatabaseSetting.Name,
		setting.DatabaseSetting.SSLMode,
		setting.DatabaseSetting.TimeZone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	DB = db
	return nil
}

func MigrateDatabase() error {
	return nil
}
