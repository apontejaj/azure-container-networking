package grpc

import (
	"context"

	pb "github.com/Azure/azure-container-networking/cns/grpc/v1alpha"
	"github.com/Azure/azure-container-networking/cns/restserver"
	"go.uber.org/zap"
)

// CNSService defines the CNS gRPC service.
type CNS struct {
	pb.UnimplementedCNSServer
	Logger *zap.Logger
	State  *restserver.HTTPRestService
}

func (s *Server) SetOrchestratorInfo(ctx context.Context, req *pb.SetOrchestratorInfoRequest) (*pb.SetOrchestratorInfoResponse, error) {
	s.Logger.Info("SetOrchestratorInfo called",
		zap.String("NodeID", req.NodeID),
		zap.String("OrchestratorType", req.OrchestratorType))
	
	// Implement your logic to handle the SetOrchestratorInfo request.
	// Assume it's successful and return a success response.
	return &pb.SetOrchestratorInfoResponse{
		ReturnCode: 0,
		Message:    "Successfully set orchestrator info",
	}, nil
}

func (s *Server) GetNodeInfo(ctx context.Context, req *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	return &pb.NodeInfoResponse{}, nil
}

func (s *CNS) UnregisterNode(ctx context.Context, req *pb.UnregisterNodeRequest) (*pb.UnregisterNodeResponse, error) {
	s.Logger.Info("Unregistering node", zap.String("nodeID", req.NodeID))
	// todo: Implement the logic
	return &pb.UnregisterNodeResponse{}, nil
}
