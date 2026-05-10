package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(serviceName string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	base, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return base.With(zap.String("service", serviceName)), nil
}
