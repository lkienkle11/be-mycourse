package setting

import (
	"os"
	"strings"

	"mycourse-io-be/pkg/envbool"
)

func expandYAMLConfig(c *yamlConfig, dotEnv map[string]string) {
	expand := func(s string) string {
		return os.Expand(s, func(key string) string {
			if v, ok := dotEnv[key]; ok {
				return v
			}
			return os.Getenv(key)
		})
	}
	expandYAMLServerDBApp(c, expand)
	expandYAMLIntegrations(c, expand)
	expandYAMLMediaSection(c, expand)
	expandYAMLLoggingSection(c, expand)
}

func expandYAMLServerDBApp(c *yamlConfig, expand func(string) string) {
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
}

func expandYAMLIntegrations(c *yamlConfig, expand func(string) string) {
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
}

func expandYAMLLoggingSection(c *yamlConfig, expand func(string) string) {
	c.Logging.Level = expand(c.Logging.Level)
	c.Logging.Format = expand(c.Logging.Format)
	c.Logging.FilePath = expand(c.Logging.FilePath)
	c.Logging.ServiceName = expand(c.Logging.ServiceName)
	c.Logging.Environment = expand(c.Logging.Environment)
	c.Logging.Version = expand(c.Logging.Version)
	c.Logging.RedirectStdlog = expand(c.Logging.RedirectStdlog)
}

func expandYAMLMediaSection(c *yamlConfig, expand func(string) string) {
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
	c.Media.BunnyStreamReadOnlyAPIKey = expand(c.Media.BunnyStreamReadOnlyAPIKey)
	c.Media.BunnyStreamLibraryID = expand(c.Media.BunnyStreamLibraryID)
	c.Media.BunnyStreamBaseURL = expand(c.Media.BunnyStreamBaseURL)
	c.Media.BunnyStreamCDNHostname = expand(c.Media.BunnyStreamCDNHostname)
	c.Media.BunnyStorageEndpoint = expand(c.Media.BunnyStorageEndpoint)
	c.Media.BunnyStoragePassword = expand(c.Media.BunnyStoragePassword)
	c.Media.LocalFileURLSecret = expand(c.Media.LocalFileURLSecret)
}

func applyYAMLToGlobals(c *yamlConfig) {
	applyYAMLServerGlobals(c)
	applyYAMLDatabaseGlobals(c)
	applyYAMLAppBrevoGlobals(c)
	applyYAMLRedisSupabaseGlobals(c)
	applyYAMLMediaGlobals(c)
	applyYAMLLoggingGlobals(c)
}

func applyYAMLLoggingGlobals(c *yamlConfig) {
	level := strings.TrimSpace(c.Logging.Level)
	if level == "" {
		level = "info"
	}
	LogSetting.Level = level

	format := strings.TrimSpace(strings.ToLower(c.Logging.Format))
	if format == "" {
		format = defaultLogFormatFromRunMode()
	}
	LogSetting.Format = format

	LogSetting.FilePath = strings.TrimSpace(c.Logging.FilePath)
	LogSetting.ServiceName = effectiveLogServiceName(c.Logging.ServiceName)
	LogSetting.Environment = effectiveLogEnvironment(c.Logging.Environment)
	LogSetting.Version = effectiveLogVersion(c.Logging.Version)
	LogSetting.RedirectStdLog = envbool.ParseTrue(c.Logging.RedirectStdlog)
}

func defaultLogFormatFromRunMode() string {
	if strings.EqualFold(strings.TrimSpace(ServerSetting.RunMode), "debug") {
		return "console"
	}
	return "json"
}

func effectiveLogServiceName(s string) string {
	s = strings.TrimSpace(s)
	if s != "" {
		return s
	}
	return "be-mycourse"
}

func effectiveLogEnvironment(s string) string {
	s = strings.TrimSpace(s)
	if s != "" {
		return s
	}
	if e := strings.TrimSpace(os.Getenv("STAGE")); e != "" {
		return e
	}
	return "default"
}

func effectiveLogVersion(s string) string {
	s = strings.TrimSpace(s)
	if s != "" {
		return s
	}
	return strings.TrimSpace(os.Getenv("APP_VERSION"))
}

func applyYAMLServerGlobals(c *yamlConfig) {
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
}

func applyYAMLDatabaseGlobals(c *yamlConfig) {
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
}

func applyYAMLAppBrevoGlobals(c *yamlConfig) {
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
}

func applyYAMLRedisSupabaseGlobals(c *yamlConfig) {
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
}

func applyYAMLMediaGlobals(c *yamlConfig) {
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
	MediaSetting.BunnyStreamReadOnlyAPIKey = c.Media.BunnyStreamReadOnlyAPIKey
	MediaSetting.BunnyStreamLibraryID = c.Media.BunnyStreamLibraryID
	MediaSetting.BunnyStreamBaseURL = c.Media.BunnyStreamBaseURL
	MediaSetting.BunnyStreamCDNHostname = c.Media.BunnyStreamCDNHostname
	MediaSetting.BunnyStorageEndpoint = c.Media.BunnyStorageEndpoint
	MediaSetting.BunnyStoragePassword = c.Media.BunnyStoragePassword
	MediaSetting.LocalFileURLSecret = c.Media.LocalFileURLSecret
}
