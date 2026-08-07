package logger

import (
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

func Init(env string, level string) {
	var core zapcore.Core

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	zapLevel := parseLevel(level, env)

	if isProduction(env) {
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   "logs/app.log",
			MaxSize:    100, // MB before rotation
			MaxBackups: 30,  // number of old files to keep
			MaxAge:     30,  // days
			Compress:   true,
		})

		consoleWriter := zapcore.AddSync(os.Stdout)

		core = zapcore.NewTee(
			zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), fileWriter, zapLevel),
			zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), consoleWriter, zapLevel),
		)
	} else {
		consoleWriter := zapcore.AddSync(os.Stdout)
		devEncoderCfg := zap.NewDevelopmentEncoderConfig()
		devEncoderCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("15:04:05.000"))
		}
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(devEncoderCfg),
			consoleWriter,
			zapLevel,
		)
	}

	Log = zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

func isProduction(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

func parseLevel(level string, env string) zapcore.Level {
	if level != "" {
		var l zapcore.Level
		if err := l.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(level)))); err == nil {
			return l
		}
	}
	if isProduction(env) {
		return zapcore.InfoLevel
	}
	return zapcore.DebugLevel
}
