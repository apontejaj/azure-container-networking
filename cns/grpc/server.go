package cns

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"go.uber.org/zap"

	pb "grpc/protogen"
)

// Server struct to hold the gRPC server settings and the CNS service.
type Server struct {
	Settings   GrpcServerSettings
	CnsService pb.CNSServer
	Logger     *zap.Logger
}

// GrpcServerSettings holds the gRPC server settings.
type GrpcServerSettings struct {
	IPAddress string
	Port      uint16s
}

// NewServer initializes a new gRPC server instance.
func NewServer(settings GrpcServerSettings, cnsService pb.CNSServer, logger *zap.Logger) (*Server, error) {
	if cnsService == nil {
		return nil, fmt.Errorf("CNS service is not defined")
	}

	server := &Server{
		Settings:   settings,
		CnsService: cnsService,
		Logger:     logger,
	}

	return server, nil
}

// Start starts the gRPC server.
func (s *Server) Start() error {
	address := net.JoinHostPort(s.Settings.IPAddress, strconv.FormatUint(uint64(s.Settings.Port), 10))
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on address %s: %w", address, err)
	}
	s.Logger.Sugar().Infof("gRPC listening on %s", address)

	grpcServer := grpc.NewServer()
	pb.RegisterCNSServer(grpcServer, s.CnsService)

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}
