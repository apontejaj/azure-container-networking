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

func (s *CNS) SetOrchestratorInfo(ctx context.Context, req *pb.SetOrchestratorInfoRequest) (*pb.SetOrchestratorInfoResponse, error) {
	s.Logger.Info("SetOrchestratorInfo called", zap.String("NodeID", req.NodeID), zap.String("OrchestratorType", req.OrchestratorType))
	// todo: Implement the logic
	return &pb.SetOrchestratorInfoResponse{}, nil
}

func (s *CNS) GetNodeInfo(ctx context.Context, req *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	// todo: Implement the logic
	return &pb.NodeInfoResponse{}, nil
}

func (s *CNS) UnregisterNode(ctx context.Context, req *pb.UnregisterNodeRequest) (*pb.UnregisterNodeResponse, error) {
	s.Logger.Info("Unregistering node", zap.String("nodeID", req.NodeID))
	// todo: Implement the logic
	return &pb.UnregisterNodeResponse{}, nil
}
