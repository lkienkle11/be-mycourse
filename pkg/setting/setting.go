package setting

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type App struct {
	JWTSecret string
	ApiKey    string
}

type Server struct {
	RunMode string
	Host    string
	Port    string
}

type Database struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}

var (
	AppSetting      = &App{}
	ServerSetting   = &Server{}
	DatabaseSetting = &Database{}
	RedisSetting    = &Redis{}
)

func Setup() error {
	cfgPath := resolveConfigPath()
	cfg, err := ini.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load ini %s: %w", cfgPath, err)
	}

	mapToServer(cfg)
	mapToDatabase(cfg)
	mapToApp(cfg)
	mapToRedis(cfg)
	return nil
}

func resolveConfigPath() string {
	if envFile := os.Getenv("APP_ENV_FILE"); envFile != "" {
		return envFile
	}

	if strings.EqualFold(os.Getenv("TEST"), "true") {
		return filepath.Clean("../config/app.ini")
	}

	stage := os.Getenv("STAGE")
	if stage == "" {
		return filepath.Clean("config/app.ini")
	}

	return filepath.Clean(fmt.Sprintf("config/app-%s.ini", stage))
}

func mapToServer(cfg *ini.File) {
	section := cfg.Section("server")
	ServerSetting.RunMode = section.Key("RunMode").MustString("debug")
	ServerSetting.Host = section.Key("Host").MustString("0.0.0.0")
	ServerSetting.Port = section.Key("Port").MustString("8080")
}

func mapToDatabase(cfg *ini.File) {
	section := cfg.Section("database")
	DatabaseSetting.Host = section.Key("Host").MustString("localhost")
	DatabaseSetting.Port = section.Key("Port").MustInt(5432)
	DatabaseSetting.User = section.Key("User").MustString("postgres")
	DatabaseSetting.Password = section.Key("Password").MustString("postgres")
	DatabaseSetting.Name = section.Key("Name").MustString("mycourse")
	DatabaseSetting.SSLMode = section.Key("SSLMode").MustString("disable")
	DatabaseSetting.TimeZone = section.Key("TimeZone").MustString("Asia/Ho_Chi_Minh")
}

func mapToApp(cfg *ini.File) {
	section := cfg.Section("app")
	AppSetting.JWTSecret = section.Key("JWTSecret").MustString("change-me")
	AppSetting.ApiKey = section.Key("ApiKey").MustString("")
}

func mapToRedis(cfg *ini.File) {
	section := cfg.Section("redis")
	RedisSetting.Addr = section.Key("Addr").MustString("localhost:6379")
	RedisSetting.Password = section.Key("Password").MustString("")
	RedisSetting.DB = section.Key("DB").MustInt(0)
}
