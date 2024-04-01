package restserver

import (
	"context"
	"fmt"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/client"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/types"
	"github.com/pkg/errors"
)

type SFSWIFTv2Middleware struct {
	CnsClient *client.Client
}

// IPConfigsRequestHandlerWrapper is the middleware function for handling SWIFT v2 IP config requests for SF standalone scenario. This function wraps the default SWIFT request
// and release IP configs handlers.
func (m *SFSWIFTv2Middleware) IPConfigsRequestHandlerWrapper(_, _ cns.IPConfigsHandlerFunc) cns.IPConfigsHandlerFunc {
	return func(ctx context.Context, req cns.IPConfigsRequest) (*cns.IPConfigsResponse, error) {
		// unmarshal & retrieve podInfo from OrchestratorContext
		podInfo, err := cns.NewPodInfoFromIPConfigsRequest(req)
		ipConfigsResp := &cns.IPConfigsResponse{
			Response: cns.Response{
				ReturnCode: types.Success,
			},
			PodIPInfo: []cns.PodIpInfo{},
		}
		if err != nil {
			ipConfigsResp.Response.ReturnCode = types.UnexpectedError
			return ipConfigsResp, errors.Wrapf(err, "Failed to receive PodInfo after unmarshalling from IPConfigsRequest %v", req)
		}

		// SwiftV2-SF will always request for secondaryInterfaces for a pod
		req.SecondaryInterfacesExist = true
		logger.Printf("[SWIFTv2Middleware] pod %s has secondary interface : %v", podInfo.Name(), req.SecondaryInterfacesExist)

		// get the IPConfig for swiftv2 SF scenario by calling into cns getNC api
		SWIFTv2PodsIPInfo, err := m.getIPConfig(ctx, podInfo)
		if err != nil {
			return &cns.IPConfigsResponse{
				Response: cns.Response{
					ReturnCode: types.FailedToAllocateIPConfig,
					Message:    fmt.Sprintf("AllocateIPConfig failed: %v, IP config request is %v", err, req),
				},
				PodIPInfo: []cns.PodIpInfo{},
			}, errors.Wrapf(err, "failed to get SWIFTv2 IP config : %v", req)
		}
		ipConfigsResp.PodIPInfo = SWIFTv2PodsIPInfo
		return ipConfigsResp, nil
	}
}

// getIPConfig returns the pod's SWIFT V2 IP configuration.
func (m *SFSWIFTv2Middleware) getIPConfig(ctx context.Context, podInfo cns.PodInfo) ([]cns.PodIpInfo, error) {
	// Create an array and return the same for the aboce functions
	podsIPInfo := []cns.PodIpInfo{}
	orchestratorContext, err := podInfo.OrchestratorContext()
	if err != nil {
		return podsIPInfo, fmt.Errorf("error getting orchestrator context from PodInfo %w", err)
	}
	// call getNC via CNSClient
	resp, err := m.CnsClient.GetAllNetworkContainers(ctx, orchestratorContext)
	if err != nil {
		return podsIPInfo, fmt.Errorf("error getNetworkContainerByOrchestrator Context %w", err)
	}

	for _, containerReponse := range resp {
		switch containerReponse.NetworkInterfaceInfo.NICType {
		case cns.DelegatedVMNIC:
			// Check if the ncstate/ipconfig ready. If one of the fields is empty, return error
			if containerReponse.IPConfiguration.IPSubnet.IPAddress == "" || containerReponse.NetworkInterfaceInfo.MACAddress == "" || containerReponse.NetworkContainerID == "" || containerReponse.IPConfiguration.GatewayIPAddress == "" {
				return podsIPInfo, fmt.Errorf("one of the fields for GetNCResponse is empty for given NC: %+v", resp) //nolint:goerr113 // return error
			}

		case cns.BackendNIC:
			if containerReponse.NetworkInterfaceInfo.MACAddress == "" {
				return podsIPInfo, fmt.Errorf("one of the fields for GetNCResponse is empty for given NC: %+v", resp)
			}

			logger.Debugf("[SWIFTv2-SF] NetworkContainerResponse for pod %s is : %+v", podInfo.Name(), resp)

			podIPInfo := cns.PodIpInfo{
				PodIPConfig:                     containerReponse.IPConfiguration.IPSubnet,
				MacAddress:                      containerReponse.NetworkInterfaceInfo.MACAddress,
				NICType:                         containerReponse.NetworkInterfaceInfo.NICType,
				SkipDefaultRoutes:               false,
				NetworkContainerPrimaryIPConfig: containerReponse.IPConfiguration,
				AddInterfacesDataToPodInfo:      true,
			}
			podsIPInfo = append(podsIPInfo, podIPInfo)
		}
	}

	return podsIPInfo, nil
}
