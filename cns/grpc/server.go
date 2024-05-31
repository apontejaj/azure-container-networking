package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/Azure/azure-container-networking/cns"
	pb "github.com/Azure/azure-container-networking/cns/grpc/v1alpha"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/restserver"
	"github.com/Azure/azure-container-networking/cns/types"
	"github.com/Azure/azure-container-networking/store"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server struct to hold the gRPC server settings and the CNS service.
type Server struct {
	Settings   ServerSettings
	CnsService pb.CNSServer
	Logger     *zap.Logger
}

// GrpcServerSettings holds the gRPC server settings.
type ServerSettings struct {
	IPAddress string
	Port      uint16
}

// CNSService defines the CNS gRPC service.
type CNS struct {
	pb.UnimplementedCNSServer
	sync.RWMutex
	Logger *zap.Logger
	*restserver.HTTPRestService
	state *restserver.HttpRestServiceState
	store store.KeyValueStore
}

// NewServer initializes a new gRPC server instance.
func NewServer(settings ServerSettings, cnsService pb.CNSServer, logger *zap.Logger) (*Server, error) {
	if cnsService == nil {
		ErrCNSServiceNotDefined := errors.New("CNS service is not defined")
		return nil, fmt.Errorf("Failed to create new gRPC server: %w", ErrCNSServiceNotDefined)
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
	pb.RegisterCNSServer(grpcServer, s.CnsService)

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

// HealthCheck is a simple method to check if the server is running.
func (s *CNS) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	s.Logger.Info("HealthCheck called")
	return &pb.HealthCheckResponse{Status: "Server is running"}, nil
}

// areNCsPresent returns true if NCs are present in CNS, false if no NCs are present.
func (s *CNS) areNCsPresent() bool {
	if len(s.state.ContainerStatus) == 0 && len(s.state.ContainerIDByOrchestratorContext) == 0 {
		return false
	}
	return true
}

// SaveState persists the current state of the service.
func (s *CNS) SaveState() error {
	// Skip if a store is not provided.
	if s.store == nil {
		s.Logger.Warn("Store not initialized.")
		return nil
	}

	// Update time stamp.
	s.state.TimeStamp = time.Now()
	err := s.store.Write("ContainerNetworkService", s.state)
	if err != nil {
		s.Logger.Error("Failed to save state", zap.Error(err))
	}

	return err
}

// SetOrchestratorInfo handles setting the orchestrator information for a node.
func (s *CNS) SetOrchestratorInfo(ctx context.Context, req *pb.SetOrchestratorInfoRequest) (*pb.SetOrchestratorInfoResponse, error) {
	s.Logger.Info("SetOrchestratorInfo called", zap.String("nodeID", req.NodeID), zap.String("orchestratorType", req.OrchestratorType))

	s.Lock()

	nodeID := s.state.NodeID

	var returnMessage string
	var returnCode types.ResponseCode

	if nodeID == "" || nodeID == req.NodeID || !s.areNCsPresent() {
		switch req.OrchestratorType {
		case cns.ServiceFabric, cns.Kubernetes, cns.KubernetesCRD, cns.WebApps, cns.Batch, cns.DBforPostgreSQL, cns.AzureFirstParty:
			s.state.OrchestratorType = req.OrchestratorType
			s.state.NodeID = req.NodeID
			logger.SetContextDetails(req.OrchestratorType, req.NodeID)
			s.SaveState()
			returnMessage = "Orchestrator information set successfully"
			returnCode = types.Success
		default:
			returnMessage = fmt.Sprintf("Invalid Orchestrator type %v", req.OrchestratorType)
			returnCode = types.UnsupportedOrchestratorType
		}
	} else {
		returnMessage = fmt.Sprintf("Invalid request since this node has already been registered as %s", nodeID)
		returnCode = types.InvalidRequest
	}

	s.Logger.Info("SetOrchestratorInfo response", zap.String("returnMessage", returnMessage), zap.Int("returnCode", int(returnCode)))
	resp := &pb.SetOrchestratorInfoResponse{}

	s.Unlock()
	return resp, nil
}

// GetNodeInfo handles retrieving detailed information about a specific node.
func (s *CNS) GetNodeInfo(ctx context.Context, req *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	s.Logger.Info("GetNodeInfo called", zap.String("nodeID", req.NodeID))

	s.RLock()
	defer s.RUnlock()

	// Simulate getting node information.
	nodeInfo := &pb.NodeInfoResponse{
		NodeID:    req.NodeID,
		Name:      "Sample",
		Ip:        "192.168.1.1",
		IsHealthy: true,
		Status:    "running",
		Message:   "Node is healthy",
	}

	s.Logger.Info("GetNodeInfo response", zap.String("nodeID", nodeInfo.NodeID), zap.String("status", nodeInfo.Status))
	return nodeInfo, nil
}
