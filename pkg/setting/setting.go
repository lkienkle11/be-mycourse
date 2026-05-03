package setting

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"

	"mycourse-io-be/pkg/envbool"
)

type App struct {
	JWTSecret          string
	ApiKey             string
	AppBaseURL         string
	CorsAllowedOrigins []string
}

type Brevo struct {
	APIKey      string
	SenderEmail string
	SenderName  string
}

type Server struct {
	RunMode string
	Host    string
	Port    string
}

// Database holds pure PostgreSQL settings (GORM / raw SQL via the same DSN).
// Prefer URL for a full connection string, or set Host + Name (and usually User) to build one.
type Database struct {
	URL      string
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}

type Supabase struct {
	ProjectRef     string
	URL            string
	AnonKey        string
	ServiceRoleKey string
	DBURL          string
}

type Media struct {
	AppMediaProvider     string
	B2KeyID              string
	B2AppKey             string
	B2Bucket             string
	B2BaseURL            string
	GcoreAPIBaseURL      string
	GcoreAPIToken        string
	GcoreCDNURL          string
	BunnyStreamAPIBase   string
	BunnyStreamAPIKey    string
	BunnyStreamLibraryID string
	BunnyStreamBaseURL   string
	BunnyStorageEndpoint string
	BunnyStoragePassword string
	LocalFileURLSecret   string
}

var (
	AppSetting      = &App{}
	ServerSetting   = &Server{}
	DatabaseSetting = &Database{}
	RedisSetting    = &Redis{}
	SupabaseSetting = &Supabase{}
	BrevoSetting    = &Brevo{}
	MediaSetting    = &Media{}
)

type yamlConfig struct {
	Server   yamlServer   `yaml:"server"`
	Database yamlDatabase `yaml:"database"`
	App      yamlApp      `yaml:"app"`
	Redis    yamlRedis    `yaml:"redis"`
	Supabase yamlSupabase `yaml:"supabase"`
	Brevo    yamlBrevo    `yaml:"brevo"`
	Media    yamlMedia    `yaml:"media"`
}

type yamlServer struct {
	RunMode string `yaml:"run_mode"`
	Host    string `yaml:"host"`
	Port    string `yaml:"port"`
}

type yamlDatabase struct {
	URL      string `yaml:"url"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"ssl_mode"`
}

type yamlCors struct {
	AllowedOrigins string `yaml:"allowed_origins"`
}

type yamlApp struct {
	JWTSecret  string   `yaml:"jwt_secret"`
	ApiKey     string   `yaml:"api_key"`
	AppBaseURL string   `yaml:"app_base_url"`
	Cors       yamlCors `yaml:"cors"`
}

type yamlBrevo struct {
	APIKey      string `yaml:"api_key"`
	SenderEmail string `yaml:"sender_email"`
	SenderName  string `yaml:"sender_name"`
}

type yamlRedis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type yamlSupabase struct {
	ProjectRef     string `yaml:"project_ref"`
	URL            string `yaml:"url"`
	AnonKey        string `yaml:"anon_key"`
	ServiceRoleKey string `yaml:"service_role_key"`
	DBURL          string `yaml:"db_url"`
}

type yamlMedia struct {
	AppMediaProvider     string `yaml:"app_media_provider"`
	B2KeyID              string `yaml:"b2_key_id"`
	B2AppKey             string `yaml:"b2_app_key"`
	B2Bucket             string `yaml:"b2_bucket"`
	B2BaseURL            string `yaml:"b2_base_url"`
	GcoreAPIBaseURL      string `yaml:"gcore_api_base_url"`
	GcoreAPIToken        string `yaml:"gcore_api_token"`
	GcoreCDNURL          string `yaml:"gcore_cdn_url"`
	BunnyStreamAPIBase   string `yaml:"bunny_stream_api_base_url"`
	BunnyStreamAPIKey    string `yaml:"bunny_stream_api_key"`
	BunnyStreamLibraryID string `yaml:"bunny_stream_library_id"`
	BunnyStreamBaseURL   string `yaml:"bunny_stream_base_url"`
	BunnyStorageEndpoint string `yaml:"bunny_storage_endpoint"`
	BunnyStoragePassword string `yaml:"bunny_storage_password"`
	LocalFileURLSecret   string `yaml:"local_file_url_secret"`
}

// Setup reads .env then .env.<STAGE> into an in-memory map (no godotenv.Load),
// loads one YAML per environment (config/app.yaml when STAGE unset and not TEST;
// config/app-<STAGE>.yaml when STAGE is set, or STAGE implied by TEST=true → test), expands ${VAR}
// map then os.Getenv, and fills the package globals.
func Setup() error {
	stage := currentStage()
	dotEnv, err := readDotEnvMaps(stage)
	if err != nil {
		return err
	}

	yamlPath := resolveYAMLPath()
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("read yaml %s: %w", yamlPath, err)
	}

	var cfg yamlConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse yaml %s: %w", yamlPath, err)
	}

	expandYAMLConfig(&cfg, dotEnv)
	applyYAMLToGlobals(&cfg)
	return nil
}

// currentStage drives which .env.<stage> is merged and (with resolveYAMLPath) which app-*.yaml is used.
// TEST=true forces stage "test" for both.
func currentStage() string {
	if envbool.Enabled("TEST") {
		return "test"
	}
	return strings.TrimSpace(os.Getenv("STAGE"))
}

// readDotEnvMaps parses .env then .env.<stage> (if stage set) into one map.
// Missing files are skipped. Values from the stage file override .env.
func readDotEnvMaps(stage string) (map[string]string, error) {
	out := make(map[string]string)
	mergeFile := func(path string) error {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		m, err := godotenv.Read(path)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		for k, v := range m {
			out[k] = v
		}
		return nil
	}
	if err := mergeFile(".env"); err != nil {
		return nil, err
	}
	if stage != "" {
		if err := mergeFile(".env." + stage); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func resolveYAMLPath() string {
	if envFile := os.Getenv("APP_ENV_FILE"); envFile != "" {
		return filepath.Clean(envFile)
	}
	stage := currentStage()
	if stage == "" {
		return filepath.Clean("config/app.yaml")
	}
	return filepath.Clean(fmt.Sprintf("config/app-%s.yaml", stage))
}

func expandYAMLConfig(c *yamlConfig, dotEnv map[string]string) {
	expand := func(s string) string {
		return os.Expand(s, func(key string) string {
			if v, ok := dotEnv[key]; ok {
				return v
			}
			return os.Getenv(key)
		})
	}

	c.Server.RunMode = expand(c.Server.RunMode)
	c.Server.Host = expand(c.Server.Host)
	c.Server.Port = expand(c.Server.Port)

	c.Database.URL = expand(c.Database.URL)
	c.Database.Host = expand(c.Database.Host)
	c.Database.Port = expand(c.Database.Port)
	c.Database.User = expand(c.Database.User)
	c.Database.Password = expand(c.Database.Password)
	c.Database.Name = expand(c.Database.Name)
	c.Database.SSLMode = expand(c.Database.SSLMode)

	c.App.JWTSecret = expand(c.App.JWTSecret)
	c.App.ApiKey = expand(c.App.ApiKey)

	c.Redis.Addr = expand(c.Redis.Addr)
	c.Redis.Password = expand(c.Redis.Password)

	c.Supabase.ProjectRef = expand(c.Supabase.ProjectRef)
	c.Supabase.URL = expand(c.Supabase.URL)
	c.Supabase.AnonKey = expand(c.Supabase.AnonKey)
	c.Supabase.ServiceRoleKey = expand(c.Supabase.ServiceRoleKey)
	c.Supabase.DBURL = expand(c.Supabase.DBURL)

	c.App.AppBaseURL = expand(c.App.AppBaseURL)
	c.App.Cors.AllowedOrigins = expand(c.App.Cors.AllowedOrigins)
	c.Brevo.APIKey = expand(c.Brevo.APIKey)
	c.Brevo.SenderEmail = expand(c.Brevo.SenderEmail)
	c.Brevo.SenderName = expand(c.Brevo.SenderName)

	c.Media.AppMediaProvider = expand(c.Media.AppMediaProvider)
	c.Media.B2KeyID = expand(c.Media.B2KeyID)
	c.Media.B2AppKey = expand(c.Media.B2AppKey)
	c.Media.B2Bucket = expand(c.Media.B2Bucket)
	c.Media.B2BaseURL = expand(c.Media.B2BaseURL)
	c.Media.GcoreAPIBaseURL = expand(c.Media.GcoreAPIBaseURL)
	c.Media.GcoreAPIToken = expand(c.Media.GcoreAPIToken)
	c.Media.GcoreCDNURL = expand(c.Media.GcoreCDNURL)
	c.Media.BunnyStreamAPIBase = expand(c.Media.BunnyStreamAPIBase)
	c.Media.BunnyStreamAPIKey = expand(c.Media.BunnyStreamAPIKey)
	c.Media.BunnyStreamLibraryID = expand(c.Media.BunnyStreamLibraryID)
	c.Media.BunnyStreamBaseURL = expand(c.Media.BunnyStreamBaseURL)
	c.Media.BunnyStorageEndpoint = expand(c.Media.BunnyStorageEndpoint)
	c.Media.BunnyStoragePassword = expand(c.Media.BunnyStoragePassword)
	c.Media.LocalFileURLSecret = expand(c.Media.LocalFileURLSecret)
}

func applyYAMLToGlobals(c *yamlConfig) {
	if strings.TrimSpace(c.Server.RunMode) != "" {
		ServerSetting.RunMode = c.Server.RunMode
	} else {
		ServerSetting.RunMode = "debug"
	}
	if strings.TrimSpace(c.Server.Host) != "" {
		ServerSetting.Host = c.Server.Host
	} else {
		ServerSetting.Host = "0.0.0.0"
	}
	if strings.TrimSpace(c.Server.Port) != "" {
		ServerSetting.Port = c.Server.Port
	} else {
		ServerSetting.Port = "8080"
	}

	DatabaseSetting.URL = c.Database.URL
	if strings.TrimSpace(c.Database.Port) != "" {
		DatabaseSetting.Port = c.Database.Port
	} else {
		DatabaseSetting.Port = "5432"
	}
	DatabaseSetting.Host = c.Database.Host
	DatabaseSetting.User = c.Database.User
	DatabaseSetting.Password = c.Database.Password
	DatabaseSetting.Name = c.Database.Name
	if strings.TrimSpace(c.Database.SSLMode) != "" {
		DatabaseSetting.SSLMode = c.Database.SSLMode
	} else {
		DatabaseSetting.SSLMode = "prefer"
	}

	if strings.TrimSpace(c.App.JWTSecret) != "" {
		AppSetting.JWTSecret = c.App.JWTSecret
	} else {
		AppSetting.JWTSecret = "change-me"
	}
	AppSetting.ApiKey = c.App.ApiKey
	AppSetting.AppBaseURL = strings.TrimRight(strings.TrimSpace(c.App.AppBaseURL), "/")
	rawOrigins := strings.TrimSpace(c.App.Cors.AllowedOrigins)
	if rawOrigins != "" {
		parts := strings.Split(rawOrigins, ",")
		origins := make([]string, 0, len(parts))
		for _, o := range parts {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
		AppSetting.CorsAllowedOrigins = origins
	} else {
		AppSetting.CorsAllowedOrigins = []string{"http://localhost:3000"}
	}

	BrevoSetting.APIKey = c.Brevo.APIKey
	BrevoSetting.SenderEmail = c.Brevo.SenderEmail
	if strings.TrimSpace(c.Brevo.SenderName) != "" {
		BrevoSetting.SenderName = c.Brevo.SenderName
	} else {
		BrevoSetting.SenderName = "MyCourse"
	}

	if strings.TrimSpace(c.Redis.Addr) != "" {
		RedisSetting.Addr = c.Redis.Addr
	} else {
		RedisSetting.Addr = "localhost:6379"
	}
	RedisSetting.Password = c.Redis.Password
	RedisSetting.DB = c.Redis.DB

	SupabaseSetting.ProjectRef = c.Supabase.ProjectRef
	SupabaseSetting.URL = c.Supabase.URL
	SupabaseSetting.AnonKey = c.Supabase.AnonKey
	SupabaseSetting.ServiceRoleKey = c.Supabase.ServiceRoleKey
	SupabaseSetting.DBURL = c.Supabase.DBURL

	MediaSetting.B2KeyID = c.Media.B2KeyID
	MediaSetting.AppMediaProvider = c.Media.AppMediaProvider
	MediaSetting.B2AppKey = c.Media.B2AppKey
	MediaSetting.B2Bucket = c.Media.B2Bucket
	MediaSetting.B2BaseURL = c.Media.B2BaseURL
	MediaSetting.GcoreAPIBaseURL = c.Media.GcoreAPIBaseURL
	MediaSetting.GcoreAPIToken = c.Media.GcoreAPIToken
	MediaSetting.GcoreCDNURL = c.Media.GcoreCDNURL
	MediaSetting.BunnyStreamAPIBase = c.Media.BunnyStreamAPIBase
	MediaSetting.BunnyStreamAPIKey = c.Media.BunnyStreamAPIKey
	MediaSetting.BunnyStreamLibraryID = c.Media.BunnyStreamLibraryID
	MediaSetting.BunnyStreamBaseURL = c.Media.BunnyStreamBaseURL
	MediaSetting.BunnyStorageEndpoint = c.Media.BunnyStorageEndpoint
	MediaSetting.BunnyStoragePassword = c.Media.BunnyStoragePassword
	MediaSetting.LocalFileURLSecret = c.Media.LocalFileURLSecret
}

// PostgresDSN returns a libpq-compatible URL for GORM/pg drivers.
// Uses URL when set; otherwise builds from Host, Name, User, Password, Port, SSLMode.
func (d *Database) PostgresDSN() string {
	if strings.TrimSpace(d.URL) != "" {
		return strings.TrimSpace(d.URL)
	}
	host := strings.TrimSpace(d.Host)
	name := strings.TrimSpace(d.Name)
	if host == "" || name == "" {
		return ""
	}
	port := strings.TrimSpace(d.Port)
	if port == "" {
		port = "5432"
	}
	ssl := strings.TrimSpace(d.SSLMode)
	if ssl == "" {
		ssl = "prefer"
	}
	u := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + strings.TrimPrefix(name, "/"),
	}
	user := strings.TrimSpace(d.User)
	if user != "" || d.Password != "" {
		u.User = url.UserPassword(user, d.Password)
	}
	q := url.Values{}
	q.Set("sslmode", ssl)
	u.RawQuery = q.Encode()
	return u.String()
}
