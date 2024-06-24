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

func (s *CNS) SetOrchestratorInfo(_ context.Context, req *pb.SetOrchestratorInfoRequest) (*pb.SetOrchestratorInfoResponse, error) {
	s.Logger.Info("SetOrchestratorInfo called", zap.String("nodeID", req.GetNodeID()), zap.String("orchestratorType", req.GetOrchestratorType()))
	// todo: Implement the logic
	return &pb.SetOrchestratorInfoResponse{}, nil
}

func (s *CNS) GetNodeInfo(ctx context.Context, req *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	s.Logger.Info("GetNodeInfo called", zap.String("nodeID", req.GetNodeID()))

	// Fetch the node information from the state
	node, err := s.State.GetNodeInfo(req.GetNodeID())
	if err != nil {
		s.Logger.Error("Failed to get node info", zap.String("nodeID", req.GetNodeID()), zap.Error(err))
		return nil, err
	}

	// Create the response based on the fetched node information
	nodeInfo := &pb.NodeInfoResponse{
		NodeID:    node.ID,
		Name:      node.Name,
		Ip:        node.IP,
		IsHealthy: node.IsHealthy,
		Status:    node.Status,
		Message:   node.Message,
	}

	return nodeInfo, nil
}
