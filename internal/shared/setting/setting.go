package setting

import (
	"fmt"
	"mycourse-io-be/internal/shared/constants"
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

// Logging holds Uber Zap / structured logging options (YAML + env expansion).
// See docs/patterns.md (logging) and pkg/logger.
type Logging struct {
	Level          string
	Format         string // "json" | "console"
	FilePath       string
	ServiceName    string
	Environment    string
	Version        string
	RedirectStdLog bool
}

type Media struct {
	AppMediaProvider          string
	B2KeyID                   string
	B2AppKey                  string
	B2Bucket                  string
	B2BaseURL                 string
	GcoreAPIBaseURL           string
	GcoreAPIToken             string
	GcoreCDNURL               string
	BunnyStreamAPIBase        string
	BunnyStreamAPIKey         string
	BunnyStreamReadOnlyAPIKey string
	BunnyStreamLibraryID      string
	BunnyStreamBaseURL        string
	BunnyStreamCDNHostname    string
	BunnyStorageEndpoint      string
	BunnyStoragePassword      string
	LocalFileURLSecret        string
}

var (
	AppSetting      = &App{}
	ServerSetting   = &Server{}
	DatabaseSetting = &Database{}
	RedisSetting    = &Redis{}
	SupabaseSetting = &Supabase{}
	BrevoSetting    = &Brevo{}
	MediaSetting    = &Media{}
	LogSetting      = &Logging{}
)

type yamlConfig struct {
	Server   yamlServer   `yaml:"server"`
	Database yamlDatabase `yaml:"database"`
	App      yamlApp      `yaml:"app"`
	Redis    yamlRedis    `yaml:"redis"`
	Supabase yamlSupabase `yaml:"supabase"`
	Brevo    yamlBrevo    `yaml:"brevo"`
	Media    yamlMedia    `yaml:"media"`
	Logging  yamlLogging  `yaml:"logging"`
}

type yamlLogging struct {
	Level          string `yaml:"level"`
	Format         string `yaml:"format"`
	FilePath       string `yaml:"file_path"`
	ServiceName    string `yaml:"service_name"`
	Environment    string `yaml:"environment"`
	Version        string `yaml:"version"`
	RedirectStdlog string `yaml:"redirect_stdlog"`
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
	AppMediaProvider          string `yaml:"app_media_provider"`
	B2KeyID                   string `yaml:"b2_key_id"`
	B2AppKey                  string `yaml:"b2_app_key"`
	B2Bucket                  string `yaml:"b2_bucket"`
	B2BaseURL                 string `yaml:"b2_base_url"`
	GcoreAPIBaseURL           string `yaml:"gcore_api_base_url"`
	GcoreAPIToken             string `yaml:"gcore_api_token"`
	GcoreCDNURL               string `yaml:"gcore_cdn_url"`
	BunnyStreamAPIBase        string `yaml:"bunny_stream_api_base_url"`
	BunnyStreamAPIKey         string `yaml:"bunny_stream_api_key"`
	BunnyStreamReadOnlyAPIKey string `yaml:"bunny_stream_read_only_api_key"`
	BunnyStreamLibraryID      string `yaml:"bunny_stream_library_id"`
	BunnyStreamBaseURL        string `yaml:"bunny_stream_base_url"`
	BunnyStreamCDNHostname    string `yaml:"bunny_stream_cdn_hostname"`
	BunnyStorageEndpoint      string `yaml:"bunny_storage_endpoint"`
	BunnyStoragePassword      string `yaml:"bunny_storage_password"`
	LocalFileURLSecret        string `yaml:"local_file_url_secret"`
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
		return fmt.Errorf(constants.MsgReadYAML, yamlPath, err)
	}

	var cfg yamlConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf(constants.MsgParseYAML, yamlPath, err)
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
			return fmt.Errorf(constants.MsgParseDotEnv, path, err)
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
