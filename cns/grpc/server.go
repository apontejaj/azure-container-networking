package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/Azure/azure-container-networking/cns"
	pb "github.com/Azure/azure-container-networking/cns/grpc/cnsv1alpha"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/restserver"
	"github.com/Azure/azure-container-networking/cns/types"
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
	State *restserver.HTTPRestService
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

// SetOrchestratorInfo handles setting the orchestrator information for a node.
func (s *CNSService) SetOrchestratorInfo(ctx context.Context, req *pb.SetOrchestratorInfoRequest) (*pb.SetOrchestratorInfoResponse, error) {
	s.Logger.Info("SetOrchestratorInfo called", zap.String("nodeID", req.NodeID), zap.String("orchestratorType", req.OrchestratorType))

	s.State.dncPartitionKey = req.DncPartitionKey
	nodeID := s.State.state.NodeID

	var returnMessage string
	var returnCode types.ResponseCode

	if nodeID == "" || nodeID == req.NodeID || !s.State.AreNCsPresent() {
		switch req.OrchestratorType {
		case cns.ServiceFabric, cns.Kubernetes, cns.KubernetesCRD, cns.WebApps, cns.Batch, cns.DBforPostgreSQL, cns.AzureFirstParty:
			s.State.state.OrchestratorType = req.OrchestratorType
			s.State.state.NodeID = req.NodeID
			logger.SetContextDetails(req.OrchestratorType, req.NodeID)
			s.State.SaveState()
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
	return resp, nil
}
