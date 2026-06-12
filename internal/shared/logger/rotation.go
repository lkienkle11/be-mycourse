package logger

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

type rotationConfig struct {
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

func sanitizeInstanceID(instanceID string) string {
	return safePathName(strings.TrimSpace(instanceID))
}

func appendInstance(base, instanceID string) string {
	inst := sanitizeInstanceID(instanceID)
	if inst == "" {
		return base
	}
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return fmt.Sprintf("%s-%s%s", name, inst, ext)
}

func newRotator(filename string, cfg rotationConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}
}
