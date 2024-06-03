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

func (s *Server) SetOrchestratorInfo(context.Context, *pb.SetOrchestratorInfoRequest) (*pb.SetOrchestratorInfoResponse, error) {
	return &pb.SetOrchestratorInfoResponse{}, nil
}

func (s *Server) GetNodeInfo(context.Context, *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	return &pb.NodeInfoResponse{}, nil
}
