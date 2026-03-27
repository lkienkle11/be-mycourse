package logger

import (
	"strings"

	"go.uber.org/zap"
)

func New(env string) (*zap.Logger, error) {
	if strings.EqualFold(env, "production") {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
