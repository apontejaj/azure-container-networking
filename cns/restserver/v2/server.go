package v2

import (
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/restserver"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Server struct {
	*restserver.HTTPRestService
}

func New(s *restserver.HTTPRestService) *Server {
	return &Server{s}
}

func (s *Server) Start(log *zap.Logger, addr string) {
	e := echo.New()
	e.HideBanner = true
	e.GET(cns.RequestIPConfig, echo.WrapHandler(restserver.NewHandlerFuncWithHistogram(s.RequestIPConfigHandler, restserver.HttpRequestLatency)))
	e.GET(cns.RequestIPConfigs, echo.WrapHandler(restserver.NewHandlerFuncWithHistogram(s.RequestIPConfigsHandler, restserver.HttpRequestLatency)))
	e.GET(cns.ReleaseIPConfig, echo.WrapHandler(restserver.NewHandlerFuncWithHistogram(s.ReleaseIPConfigHandler, restserver.HttpRequestLatency)))
	e.GET(cns.ReleaseIPConfigs, echo.WrapHandler(restserver.NewHandlerFuncWithHistogram(s.ReleaseIPConfigsHandler, restserver.HttpRequestLatency)))
	e.GET(cns.PathDebugIPAddresses, echo.WrapHandler(http.HandlerFunc(s.HandleDebugIPAddresses)))
	e.GET(cns.PathDebugPodContext, echo.WrapHandler(http.HandlerFunc(s.HandleDebugPodContext)))
	e.GET(cns.PathDebugRestData, echo.WrapHandler(http.HandlerFunc(s.HandleDebugRestData)))
	if err := e.Start(addr); err != nil {
		log.Error("failed to run server", zap.Error(err))
	}
}
