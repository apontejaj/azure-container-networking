package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/Azure/azure-container-networking/cns/grpc/v1alpha"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	address = "127.0.0.1"
	port    = 8080
)

func main() {
	// Target server address
	target := fmt.Sprintf("%v:%d", address, port)

	// Create a connection to the gRPC server
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// Create a new CNS client
	client := pb.NewCNSClient(conn)

	// Set up the context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Request to check server health
	healthCheckRequest := &pb.HealthCheckRequest{}

	// Make the gRPC call to HealthCheck
	resp, err := client.HealthCheck(ctx, healthCheckRequest)
	if err != nil {
		log.Fatalf("failed to check health: %v", err)
	}

	log.Printf("HealthCheck response: %v", resp.Status)

	// Request to get node info
	nodeInfoRequest := &pb.NodeInfoRequest{
		NodeID: "Node123",
	}

	// Make the gRPC call to GetNodeInfo
	nodeInfoResp, err := client.GetNodeInfo(ctx, nodeInfoRequest)
	if err != nil {
		log.Fatalf("failed to get node info: %v", err)
	}

	log.Printf("GetNodeInfo response: NodeID=%v, Name=%v, IP=%v, IsHealthy=%v, Status=%v, Message=%v",
		nodeInfoResp.NodeID, nodeInfoResp.Name, nodeInfoResp.Ip, nodeInfoResp.IsHealthy, nodeInfoResp.Status, nodeInfoResp.Message)

	// Request to set orchestrator info
	// orchestratorRequest := &pb.SetOrchestratorInfoRequest{
	// 	DncPartitionKey:  "examplePartitionKey",
	// 	NodeID:           "exampleNodeID",
	// 	OrchestratorType: "Kubernetes",
	// }

	// Make the gRPC call to SetOrchestratorInfo
	// resp, err := client.SetOrchestratorInfo(ctx, orchestratorRequest)
	// if err != nil {
	// 	log.Fatalf("failed to set orchestrator info: %v", err)
	// }

	// log.Printf("SetOrchestratorInfo response: %v", resp)
}
