package mq

import (
	"testing"

	"mycourse-io-be/internal/shared/setting"
)

func TestAvailableWithoutSetup(t *testing.T) {
	if Available() {
		t.Fatal("expected no connection before SetupLavinMQ")
	}
}

func TestSetupLavinMQDisabled(t *testing.T) {
	setting.LavinMQSetting.Enabled = false
	setting.LavinMQSetting.URL = "amqp://guest:guest@127.0.0.1:5672"
	SetupLavinMQ()
	if Available() {
		t.Fatal("expected disabled lavinmq to skip connection")
	}
}

func TestSetupLavinMQEmptyURL(t *testing.T) {
	setting.LavinMQSetting.Enabled = true
	setting.LavinMQSetting.URL = ""
	SetupLavinMQ()
	if Available() {
		t.Fatal("expected empty URL to skip connection")
	}
}
