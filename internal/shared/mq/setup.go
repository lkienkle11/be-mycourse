package mq

import (
	"strings"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"mycourse-io-be/internal/shared/setting"
)

var (
	conn   *amqp.Connection
	connMu sync.RWMutex
)

// Available reports whether SetupLavinMQ opened a live connection.
func Available() bool {
	connMu.RLock()
	defer connMu.RUnlock()
	return conn != nil
}

// SetupLavinMQ dials CloudAMQP / LavinMQ when enabled and URL is set.
// Missing URL or disabled config is non-fatal (same pattern as Redis warn-on-fail).
func SetupLavinMQ() {
	if !setting.LavinMQSetting.Enabled {
		zap.L().Info("lavinmq skipped", zap.String("reason", "disabled"))
		return
	}
	url := strings.TrimSpace(setting.LavinMQSetting.URL)
	if url == "" {
		zap.L().Info("lavinmq skipped", zap.String("reason", "empty CLOUDAMQP_URL"))
		return
	}

	c, err := amqp.Dial(url)
	if err != nil {
		zap.L().Warn("lavinmq not ready", zap.Error(err))
		return
	}

	connMu.Lock()
	conn = c
	connMu.Unlock()

	zap.L().Info("lavinmq connected",
		zap.String("exchange", setting.LavinMQSetting.Exchange),
	)
}

// Close shuts down the shared LavinMQ connection.
func Close() error {
	connMu.Lock()
	defer connMu.Unlock()
	if conn == nil {
		return nil
	}
	err := conn.Close()
	conn = nil
	return err
}

func connection() *amqp.Connection {
	connMu.RLock()
	defer connMu.RUnlock()
	return conn
}
