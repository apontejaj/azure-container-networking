package zaplog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	cniLogFile       = LogPath + "azure-vnet.log"
	ipamLogFile      = LogPath + "azure-vnet-ipam.log"
	telemetryLogFile = LogPath + "azure-vnet-telemetry"
)

const (
	maxLogFileSizeInMb = 5
	maxLogFileCount    = 8
)

var logFileCNIWriter = zapcore.AddSync(&lumberjack.Logger{
	Filename:   LogPath + cniLogFile,
	MaxSize:    maxLogFileSizeInMb,
	MaxBackups: maxLogFileCount,
})

var logFileIpamWriter = zapcore.AddSync(&lumberjack.Logger{
	Filename:   LogPath + ipamLogFile,
	MaxSize:    maxLogFileSizeInMb,
	MaxBackups: maxLogFileCount,
})

var logFileTelemetryWriter = zapcore.AddSync(&lumberjack.Logger{
	Filename:   LogPath + telemetryLogFile,
	MaxSize:    maxLogFileSizeInMb,
	MaxBackups: maxLogFileCount,
})

func initZapCNILog() *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
	logLevel := zapcore.DebugLevel

	core := zapcore.NewCore(jsonEncoder, logFileCNIWriter, logLevel)
	Logger := zap.New(core)
	return Logger
}

func initIpamLog() *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
	logLevel := zapcore.DebugLevel

	core := zapcore.NewCore(jsonEncoder, logFileIpamWriter, logLevel)
	Logger := zap.New(core)
	return Logger
}

func initTelemetryLog() *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
	logLevel := zapcore.DebugLevel

	core := zapcore.NewCore(jsonEncoder, logFileTelemetryWriter, logLevel)
	Logger := zap.New(core)
	return Logger
}

func InitializeCNILogger() *zap.Logger {
	defaultCNILogger := initZapCNILog()
	return defaultCNILogger
}

func InitializeIpamLogger() *zap.Logger {
	defaultIpamLogger := initIpamLog()
	return defaultIpamLogger
}

func InitializeTelemetryLogger() *zap.Logger {
	defaultTelemetryLogger := initTelemetryLog()
	return defaultTelemetryLogger
}
