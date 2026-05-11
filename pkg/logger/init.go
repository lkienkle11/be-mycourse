package logger

import (
	"fmt"
	"os"
	"strings"

	"mycourse-io-be/pkg/setting"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Options configures zap cores (console/json, optional file tee, global fields).
type Options struct {
	Level          string
	Format         string // "json" | "console"
	FilePath       string
	ServiceName    string
	Environment    string
	Version        string
	RedirectStdLog bool
}

// InitFromSettings builds the root logger from setting.LogSetting, replaces zap globals,
// and optionally redirects the standard library log package.
// Call once after setting.Setup(); defer logger.Sync() on process shutdown.
func InitFromSettings() (*zap.Logger, error) {
	cfg := Options{
		Level:          strings.TrimSpace(setting.LogSetting.Level),
		Format:         strings.TrimSpace(setting.LogSetting.Format),
		FilePath:       strings.TrimSpace(setting.LogSetting.FilePath),
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
	stdoutCore := zapcore.NewCore(encoderForFormat(fmtNorm), zapcore.AddSync(os.Stdout), level)
	cores := []zapcore.Core{stdoutCore}
	if err := appendJSONFileCoreIfConfigured(&cores, level, opts.FilePath); err != nil {
		return nil, err
	}
	core := zapcore.NewTee(cores...)
	log := zap.New(core, zap.AddCaller(), zap.Fields(globalFields(opts)...))
	zap.ReplaceGlobals(log)
	if opts.RedirectStdLog {
		_ = zap.RedirectStdLog(log)
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
}
