package pkg

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(env string) {
	var cfg zap.Config
	if env == "prod" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "time"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	var err error
	Logger, err = cfg.Build()
	if err != nil {
		os.Stderr.WriteString("zap init failed: " + err.Error())
		Logger = zap.NewNop()
	}
}

func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
