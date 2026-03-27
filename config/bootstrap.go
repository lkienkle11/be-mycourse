package config

import apiConfig "mycourse-io-be/api/config"

func InitSystem() {
	_ = apiConfig.DefaultPermissions()
}
