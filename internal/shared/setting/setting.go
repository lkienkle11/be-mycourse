package setting

import (
	"fmt"
	"mycourse-io-be/internal/shared/constants"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"

	"mycourse-io-be/internal/shared/parsebool"
)

var postgresSchemaNameRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type App struct {
	JWTSecret          string
	ApiKey             string
	AppBaseURL         string
	AppClientBaseURL   string
	AuthCookieDomain   string
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
	URL        string
	Host       string
	Port       string
	User       string
	Password   string
	Name       string
	SSLMode    string
	SchemaName string // from SCHEMA_NAME_APP via config database.schema_name
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}

// LavinMQ holds CloudAMQP LavinMQ (RabbitMQ-compatible) connection settings.
// URL should be the AMQP URL from the CloudAMQP console (env CLOUDAMQP_URL).
type LavinMQ struct {
	URL      string
	Exchange string
	Enabled  bool
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
	LogDir         string
	AppName        string
	Vendor         string
	PathMode       string
	FileEnabled    bool
	ConsoleAlso    bool
	MaxSizeMB      int
	MaxBackups     int
	MaxAgeDays     int
	Compress       bool
	InstanceID     string
	ServiceName    string
	Environment    string
	Version        string
	RedirectStdLog bool
}

// Resilience holds circuit breaker and load-shedding thresholds (YAML resilience: block).
type Resilience struct {
	DBProbeIntervalSec     int
	DBFailuresToOpen       int
	MaxInFlight            int
	HalfOpenProbeQuota     int
	OpenCooldownSec        int
	ErrorWindowSec         int
	ErrorCountToOpen       int
	DegradedAttemptsFactor float64
}

// MediaR2Storage holds Cloudflare R2 S3-compatible credentials and public CDN base URL.
type MediaR2Storage struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string
	Region          string
	PublicURL       string
}

type Media struct {
	AppMediaProvider          string
	R2                        MediaR2Storage
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

// OAuth holds external identity provider credentials (env-expanded from config/app.yaml).
type OAuth struct {
	GoogleClientID      string
	GoogleClientSecret  string
	XClientID           string
	XClientSecret       string
	XCallbackURL        string
	DiscordClientID     string
	DiscordClientSecret string
	DiscordCallbackURL  string
}

var (
	AppSetting        = &App{}
	ServerSetting     = &Server{}
	DatabaseSetting   = &Database{}
	RedisSetting      = &Redis{}
	LavinMQSetting    = &LavinMQ{}
	SupabaseSetting   = &Supabase{}
	BrevoSetting      = &Brevo{}
	MediaSetting      = &Media{}
	OAuthSetting      = &OAuth{}
	LogSetting        = &Logging{}
	ResilienceSetting = &Resilience{}
)

type yamlConfig struct {
	Server     yamlServer     `yaml:"server"`
	Database   yamlDatabase   `yaml:"database"`
	App        yamlApp        `yaml:"app"`
	Redis      yamlRedis      `yaml:"redis"`
	LavinMQ    yamlLavinMQ    `yaml:"lavinmq"`
	Supabase   yamlSupabase   `yaml:"supabase"`
	Brevo      yamlBrevo      `yaml:"brevo"`
	Media      yamlMedia      `yaml:"media"`
	OAuth      yamlOAuth      `yaml:"oauth"`
	Logging    yamlLogging    `yaml:"logging"`
	Resilience yamlResilience `yaml:"resilience"`
}

type yamlLogging struct {
	Level          string `yaml:"level"`
	Format         string `yaml:"format"`
	FilePath       string `yaml:"file_path"`
	LogDir         string `yaml:"log_dir"`
	AppName        string `yaml:"app_name"`
	Vendor         string `yaml:"vendor"`
	PathMode       string `yaml:"path_mode"`
	FileEnabled    string `yaml:"file_enabled"`
	ConsoleAlso    string `yaml:"console_also"`
	MaxSizeMB      string `yaml:"max_size_mb"`
	MaxBackups     string `yaml:"max_backups"`
	MaxAgeDays     string `yaml:"max_age_days"`
	Compress       string `yaml:"compress"`
	InstanceID     string `yaml:"instance_id"`
	ServiceName    string `yaml:"service_name"`
	Environment    string `yaml:"environment"`
	Version        string `yaml:"version"`
	RedirectStdlog string `yaml:"redirect_stdlog"`
}

type yamlResilience struct {
	DBProbeIntervalSec     int     `yaml:"db_probe_interval_sec"`
	DBFailuresToOpen       int     `yaml:"db_failures_to_open"`
	MaxInFlight            int     `yaml:"max_in_flight"`
	HalfOpenProbeQuota     int     `yaml:"half_open_probe_quota"`
	OpenCooldownSec        int     `yaml:"open_cooldown_sec"`
	ErrorWindowSec         int     `yaml:"error_window_sec"`
	ErrorCountToOpen       int     `yaml:"error_count_to_open"`
	DegradedAttemptsFactor float64 `yaml:"degraded_attempts_factor"`
}

type yamlServer struct {
	RunMode string `yaml:"run_mode"`
	Host    string `yaml:"host"`
	Port    string `yaml:"port"`
}

type yamlDatabase struct {
	URL        string `yaml:"url"`
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	Name       string `yaml:"name"`
	SSLMode    string `yaml:"ssl_mode"`
	SchemaName string `yaml:"schema_name"`
}

type yamlCors struct {
	AllowedOrigins string `yaml:"allowed_origins"`
}

type yamlApp struct {
	JWTSecret        string   `yaml:"jwt_secret"`
	ApiKey           string   `yaml:"api_key"`
	AppBaseURL       string   `yaml:"app_base_url"`
	AppClientBaseURL string   `yaml:"app_client_base_url"`
	AuthCookieDomain string   `yaml:"auth_cookie_domain"`
	Cors             yamlCors `yaml:"cors"`
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

type yamlLavinMQ struct {
	URL      string `yaml:"url"`
	Exchange string `yaml:"exchange"`
	Enabled  string `yaml:"enabled"`
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
	R2AccountID               string `yaml:"r2_account_id"`
	R2AccessKeyID             string `yaml:"r2_access_key_id"`
	R2SecretAccessKey         string `yaml:"r2_secret_access_key"`
	R2Bucket                  string `yaml:"r2_bucket"`
	R2Endpoint                string `yaml:"r2_endpoint"`
	R2Region                  string `yaml:"r2_region"`
	R2PublicURL               string `yaml:"r2_public_url"`
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

type yamlOAuth struct {
	GoogleClientID      string `yaml:"google_client_id"`
	GoogleClientSecret  string `yaml:"google_client_secret"`
	XClientID           string `yaml:"x_client_id"`
	XClientSecret       string `yaml:"x_client_secret"`
	XCallbackURL        string `yaml:"x_callback_url"`
	DiscordClientID     string `yaml:"discord_client_id"`
	DiscordClientSecret string `yaml:"discord_client_secret"`
	DiscordCallbackURL  string `yaml:"discord_callback_url"`
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
	if parsebool.EnvEnabled("TEST") {
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

// AppSchemaName returns the PostgreSQL schema for app tables (GORM search_path, golang-migrate).
// Defaults to constants.PostgresSchemaDefault when SCHEMA_NAME_APP / database.schema_name is empty.
func (d *Database) AppSchemaName() string {
	if s := strings.TrimSpace(d.SchemaName); s != "" {
		return s
	}
	return constants.PostgresSchemaDefault
}

// EnsureAppSchemaName returns AppSchemaName or an error when the configured name is not a safe identifier.
func (d *Database) EnsureAppSchemaName() (string, error) {
	schema := d.AppSchemaName()
	if !postgresSchemaNameRE.MatchString(schema) {
		return "", fmt.Errorf("invalid database schema name %q (SCHEMA_NAME_APP)", schema)
	}
	return schema, nil
}

// OAuthGoogleConfigured reports whether Google OAuth env vars are set.
func OAuthGoogleConfigured() bool {
	o := OAuthSetting
	return strings.TrimSpace(o.GoogleClientID) != "" &&
		strings.TrimSpace(o.GoogleClientSecret) != ""
}

// OAuthXConfigured reports whether all X OAuth env vars are set.
func OAuthXConfigured() bool {
	o := OAuthSetting
	return strings.TrimSpace(o.XClientID) != "" &&
		strings.TrimSpace(o.XClientSecret) != "" &&
		strings.TrimSpace(o.XCallbackURL) != ""
}

// OAuthDiscordConfigured reports whether all Discord OAuth env vars are set.
func OAuthDiscordConfigured() bool {
	o := OAuthSetting
	return strings.TrimSpace(o.DiscordClientID) != "" &&
		strings.TrimSpace(o.DiscordClientSecret) != "" &&
		strings.TrimSpace(o.DiscordCallbackURL) != ""
}
