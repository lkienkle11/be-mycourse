package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mycourse-io-be/internal/shared/setting"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Options configures zap cores (console/json, optional file tee, global fields).
type Options struct {
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

var (
	accessMu      sync.RWMutex
	accessLogger  *zap.Logger
	legacyFileMu  sync.Mutex
	legacyLogFile *os.File
)

// InitFromSettings builds the root logger from setting.LogSetting, replaces zap globals,
// and optionally redirects the standard library log package.
// Call once after setting.Setup(); defer logger.Sync() on process shutdown.
func InitFromSettings() (*zap.Logger, error) {
	cfg := Options{
		Level:          strings.TrimSpace(setting.LogSetting.Level),
		Format:         strings.TrimSpace(setting.LogSetting.Format),
		FilePath:       strings.TrimSpace(setting.LogSetting.FilePath),
		LogDir:         strings.TrimSpace(setting.LogSetting.LogDir),
		AppName:        strings.TrimSpace(setting.LogSetting.AppName),
		Vendor:         strings.TrimSpace(setting.LogSetting.Vendor),
		PathMode:       strings.TrimSpace(setting.LogSetting.PathMode),
		FileEnabled:    setting.LogSetting.FileEnabled,
		ConsoleAlso:    setting.LogSetting.ConsoleAlso,
		MaxSizeMB:      setting.LogSetting.MaxSizeMB,
		MaxBackups:     setting.LogSetting.MaxBackups,
		MaxAgeDays:     setting.LogSetting.MaxAgeDays,
		Compress:       setting.LogSetting.Compress,
		InstanceID:     strings.TrimSpace(setting.LogSetting.InstanceID),
		ServiceName:    strings.TrimSpace(setting.LogSetting.ServiceName),
		Environment:    strings.TrimSpace(setting.LogSetting.Environment),
		Version:        strings.TrimSpace(setting.LogSetting.Version),
		RedirectStdLog: setting.LogSetting.RedirectStdLog,
	}
	return Init(cfg)
}

// Init configures zap.ReplaceGlobals and returns the same logger.
func Init(opts Options) (*zap.Logger, error) {
	level := parseZapLevel(opts.Level)
	fmtNorm := strings.ToLower(strings.TrimSpace(opts.Format))
	legacyFilePath := strings.TrimSpace(opts.FilePath)
	legacyMode := legacyFilePath != ""

	stdoutCore := zapcore.NewCore(encoderForFormat(fmtNorm), zapcore.AddSync(os.Stdout), level)
	appCores := make([]zapcore.Core, 0, 3)
	accessCores := make([]zapcore.Core, 0, 3)
	if legacyMode || !opts.FileEnabled || opts.ConsoleAlso {
		appCores = append(appCores, stdoutCore)
		accessCores = append(accessCores, stdoutCore)
	}
	if err := appendJSONFileCoreIfConfigured(&appCores, level, legacyFilePath); err != nil {
		return nil, err
	}

	startupFields := make([]zap.Field, 0, 3)
	if opts.FileEnabled && !legacyMode {
		paths, err := appendDualFileCores(&appCores, &accessCores, level, opts)
		if err != nil {
			return nil, err
		}
		startupFields = append(
			startupFields,
			zap.String("log_dir", paths.LogDir),
			zap.String("app_log", paths.AppLog),
			zap.String("access_log", paths.AccessLog),
		)
	}

	appCore := zapcore.NewTee(appCores...)
	appFields := append(globalFields(opts), zap.String("log_file", "app"))
	log := zap.New(appCore, zap.AddCaller(), zap.Fields(appFields...))
	zap.ReplaceGlobals(log)
	if legacyMode {
		setAccessLogger(log)
	} else {
		accessCore := zapcore.NewTee(accessCores...)
		accessFields := append(globalFields(opts), zap.String("log_file", "access"))
		setAccessLogger(zap.New(accessCore, zap.AddCaller(), zap.Fields(accessFields...)))
	}

	if opts.RedirectStdLog {
		_ = zap.RedirectStdLog(log)
	}
	if len(startupFields) > 0 {
		log.Info("logger_file_sink_enabled", startupFields...)
	}
	return log, nil
}

func appendJSONFileCoreIfConfigured(cores *[]zapcore.Core, level zapcore.LevelEnabler, filePath string) error {
	fp := strings.TrimSpace(filePath)
	if fp == "" {
		return nil
	}
	f, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("logger: open log file %q: %w", fp, err)
	}
	legacyFileMu.Lock()
	if legacyLogFile != nil {
		_ = legacyLogFile.Close()
	}
	legacyLogFile = f
	legacyFileMu.Unlock()
	// NDJSON file sink: one JSON object per line for Filebeat/Logstash/Elastic (infra out of repo).
	fileCfg := zap.NewProductionEncoderConfig()
	fileCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEnc := zapcore.NewJSONEncoder(fileCfg)
	*cores = append(*cores, zapcore.NewCore(fileEnc, zapcore.AddSync(f), level))
	return nil
}

func globalFields(opts Options) []zap.Field {
	var fields []zap.Field
	if opts.ServiceName != "" {
		fields = append(fields, zap.String("service", opts.ServiceName))
	}
	if opts.Environment != "" {
		fields = append(fields, zap.String("env", opts.Environment))
	}
	if opts.Version != "" {
		fields = append(fields, zap.String("version", opts.Version))
	}
	return fields
}

type dualFilePaths struct {
	LogDir    string
	AppLog    string
	AccessLog string
}

func appendDualFileCores(appCores, accessCores *[]zapcore.Core, level zapcore.LevelEnabler, opts Options) (dualFilePaths, error) {
	logDir, err := ResolveLogDir(
		PathConfig{
			AppName:  opts.AppName,
			Vendor:   opts.Vendor,
			PathMode: opts.PathMode,
		},
		opts.LogDir,
	)
	if err != nil {
		return dualFilePaths{}, err
	}
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return dualFilePaths{}, fmt.Errorf("logger: create log directory %q: %w", logDir, err)
	}

	rotation := rotationConfig{
		MaxSizeMB:  opts.MaxSizeMB,
		MaxBackups: opts.MaxBackups,
		MaxAgeDays: opts.MaxAgeDays,
		Compress:   opts.Compress,
	}
	appPath := appendInstance(filepath.Join(logDir, "app.log"), opts.InstanceID)
	accessPath := appendInstance(filepath.Join(logDir, "access.log"), opts.InstanceID)

	appEnc := zapcore.NewJSONEncoder(lokiEncoderConfig())
	accessEnc := zapcore.NewJSONEncoder(lokiEncoderConfig())
	*appCores = append(*appCores, zapcore.NewCore(appEnc, zapcore.AddSync(newRotator(appPath, rotation)), level))
	*accessCores = append(*accessCores, zapcore.NewCore(accessEnc, zapcore.AddSync(newRotator(accessPath, rotation)), level))

	return dualFilePaths{
		LogDir:    logDir,
		AppLog:    appPath,
		AccessLog: accessPath,
	}, nil
}

func encoderForFormat(format string) zapcore.Encoder {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		cfg := zap.NewProductionEncoderConfig()
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		return zapcore.NewJSONEncoder(cfg)
	default:
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(cfg)
	}
}

func parseZapLevel(s string) zapcore.LevelEnabler {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "warn", "warning":
		return zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "dpanic", "panic", "fatal":
		return zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		return zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
}

// Sync flushes any buffered log entries; safe to call on shutdown (ignores common stdout EINVAL).
func Sync() {
	_ = zap.L().Sync()
	accessMu.RLock()
	acc := accessLogger
	accessMu.RUnlock()
	if acc != nil {
		_ = acc.Sync()
	}
	legacyFileMu.Lock()
	if legacyLogFile != nil {
		_ = legacyLogFile.Close()
		legacyLogFile = nil
	}
	legacyFileMu.Unlock()
}

// Access returns the dedicated access logger. It falls back to zap.L() when not initialized.
func Access() *zap.Logger {
	accessMu.RLock()
	acc := accessLogger
	accessMu.RUnlock()
	if acc != nil {
		return acc
	}
	return zap.L()
}

func setAccessLogger(l *zap.Logger) {
	accessMu.Lock()
	accessLogger = l
	accessMu.Unlock()
}
