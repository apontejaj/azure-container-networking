package grpc

// import (
// 	"context"
// 	"fmt"
// 	"testing"

// 	"github.com/Azure/azure-container-networking/cns"
// 	pb "github.com/Azure/azure-container-networking/cns/grpc/cnsv1alpha"
// 	"github.com/Azure/azure-container-networking/cns/restserver"
// 	"github.com/Azure/azure-container-networking/cns/types"
// 	"github.com/stretchr/testify/assert"
// 	"go.uber.org/zap"
// )

// // MockKeyValueStore is a mock implementation of the KeyValueStore interface for testing purposes.
// type MockKeyValueStore struct {
// 	store map[string]interface{}
// }

// func NewMockKeyValueStore() *MockKeyValueStore {
// 	return &MockKeyValueStore{
// 		store: make(map[string]interface{}),
// 	}
// }

// func (m *MockKeyValueStore) Write(key string, value interface{}) error {
// 	m.store[key] = value
// 	return nil
// }

// func (m *MockKeyValueStore) Read(key string, value interface{}) error {
// 	if v, ok := m.store[key]; ok {
// 		// Type assertion to match the expected type of value
// 		ptr, ok := value.(*interface{})
// 		if !ok {
// 			return fmt.Errorf("value should be a pointer to interface{}")
// 		}
// 		*ptr = v
// 		return nil
// 	}
// 	return fmt.Errorf("key not found")
// }

// func (m *MockKeyValueStore) Delete(key string) error {
// 	delete(m.store, key)
// 	return nil
// }

// func (m *MockKeyValueStore) Exists(key string) bool {
// 	_, ok := m.store[key]
// 	return ok
// }

// func TestSetOrchestratorInfo(t* testing.T) {
// 	logger, _ := zap.NewDevelopment()
// 	state := &restserver.HttpRestServiceState{}
// 	// mockStore := NewMockKeyValueStore()

// 	cnsService := &CNS{
// 		Logger:          logger,
// 		HTTPRestService: &restserver.HTTPRestService{},
// 		state:           state,
// 		// store:           mockStore,
// 	}

// 	tests := []struct {
// 		name            string
// 		req             *pb.SetOrchestratorInfoRequest
// 		expectedMessage string
// 		expectedCode    types.ResponseCode
// 	}{
// 		{
// 			name: "ValidOrchestratorType",
// 			req: &pb.SetOrchestratorInfoRequest{
// 				DncPartitionKey:  "partitionKey1",
// 				NodeID:           "node1",
// 				OrchestratorType: cns.Kubernetes,
// 			},
// 			expectedMessage: "Orchestrator information set successfully",
// 			expectedCode:    types.Success,
// 		},
// 		{
// 			name: "InvalidOrchestratorType",
// 			req: &pb.SetOrchestratorInfoRequest{
// 				DncPartitionKey:  "partitionKey1",
// 				NodeID:           "node1",
// 				OrchestratorType: "InvalidType",
// 			},
// 			expectedMessage: "Invalid Orchestrator type InvalidType",
// 			expectedCode:    types.UnsupportedOrchestratorType,
// 		},
// 		{
// 			name: "NodeAlreadyRegistered",
// 			req: &pb.SetOrchestratorInfoRequest{
// 				DncPartitionKey:  "partitionKey1",
// 				NodeID:           "node2",
// 				OrchestratorType: cns.Kubernetes,
// 			},
// 			expectedMessage: "Invalid request since this node has already been registered as node1",
// 			expectedCode:    types.InvalidRequest,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Simulate existing state for the "NodeAlreadyRegistered" test case
// 			if tt.name == "NodeAlreadyRegistered" {
// 				cnsService.state.NodeID = "node1"
// 			} else {
// 				cnsService.state.NodeID = ""
// 			}

// 			resp, err := cnsService.SetOrchestratorInfo(context.Background(), tt.req)
// 			assert.NoError(t, err)
// 			assert.NotNil(t, resp)

// 			// Verify the log message and return code
// 			assert.Equal(t, tt.req.OrchestratorType, cnsService.state.OrchestratorType)
// 			assert.Equal(t, tt.req.NodeID, cnsService.state.NodeID)
// 		})
// 	}
// }

