package log

import (
	"os"

	"github.com/Azure/azure-container-networking/zaplog"
	"go.uber.org/zap"
)

var CNILogger = zaplog.InitializeCNILogger().With(zap.Int("pid", os.Getpid())).With(zap.String("component", "cni"))
