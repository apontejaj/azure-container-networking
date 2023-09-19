package log

import (
	"os"

	"github.com/Azure/azure-container-networking/zaplog"
	"go.uber.org/zap"
)

func InitializeMock() {
	zaplog.InitializeCNILogger().With(zap.Int("pid", os.Getpid())).With(zap.String("component", "cni"))
}
