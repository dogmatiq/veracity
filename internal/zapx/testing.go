package zapx

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewTesting returns a logger with concise output for use in tests.
func NewTesting(name string) *zap.Logger {
	cfg := zap.NewDevelopmentConfig()

	cfg.DisableCaller = true
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.TimeKey = zapcore.OmitKey

	return zap.Must(cfg.Build()).Named(name)
}
