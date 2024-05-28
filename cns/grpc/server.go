package grpc

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/Azure/azure-container-networking/cns/grpc/cnsv1alpha"
)

// Server struct to hold the gRPC server settings and the CNS service.
type Server struct {
	Settings   GrpcServerSettings
	CnsService pb.CNSServiceServer
	Logger     *zap.Logger
}

// GrpcServerSettings holds the gRPC server settings.
type GrpcServerSettings struct {
	IPAddress string
	Port      uint16
}

// CNSService defines the CNS gRPC service.
type CNSService struct {
	pb.UnimplementedCNSServiceServer
	Logger *zap.Logger
}

// NewServer initializes a new gRPC server instance.
func NewServer(settings GrpcServerSettings, cnsService pb.CNSServiceServer, logger *zap.Logger) (*Server, error) {
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
		log.Printf("[Listener] Failed to listen on gRPC endpoint: %+v", err)
		return fmt.Errorf("failed to listen on address %s: %w", address, err)
	}
	log.Printf("[Listener] Started listening on gRPC endpoint %s.", address)

	grpcServer := grpc.NewServer()
	pb.RegisterCNSServiceServer(grpcServer, s.CnsService)

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

// Set Orchestratr 
// Status Health
