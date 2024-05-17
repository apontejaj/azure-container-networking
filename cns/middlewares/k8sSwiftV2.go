package middlewares

import (
	"context"
	"fmt"
	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/configuration"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/middlewares/utils"
	"github.com/Azure/azure-container-networking/cns/types"
	"github.com/Azure/azure-container-networking/crd/multitenancy/api/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errMTPNCNotReady            = errors.New("mtpnc is not ready")
	errInvalidSWIFTv2NICType    = errors.New("invalid NIC type for SWIFT v2 scenario")
	errInvalidMTPNCPrefixLength = errors.New("invalid prefix length for MTPNC primaryIP, must be 32")
)

const (
	prefixLength     = 32
	overlayGatewayv4 = "169.254.1.1"
	virtualGW        = "169.254.2.1"
	overlayGatewayV6 = "fe80::1234:5678:9abc"
)

type K8sSWIFTv2Middleware struct {
	Cli client.Client
}

// Verify interface compliance at compile time
var _ cns.IPConfigsHandlerMiddleware = (*K8sSWIFTv2Middleware)(nil)

// IPConfigsRequestHandlerWrapper is the middleware function for handling SWIFT v2 IP configs requests for AKS-SWIFT. This function wrapped the default SWIFT request
// and release IP configs handlers.
func (k *K8sSWIFTv2Middleware) IPConfigsRequestHandlerWrapper(defaultHandler, failureHandler cns.IPConfigsHandlerFunc) cns.IPConfigsHandlerFunc {
	return func(ctx context.Context, req cns.IPConfigsRequest) (*cns.IPConfigsResponse, error) {
<<<<<<< HEAD
		podInfo, respCode, message := k.validateIPConfigsRequest(ctx, &req)
=======
		podInfo, respCode, message := m.validateIPConfigsRequest(ctx, &req)
		// Add a function to fetch the macaddresses
>>>>>>> 496b4e3f (Handling multiple interface information)

		if respCode != types.Success {
			return &cns.IPConfigsResponse{
				Response: cns.Response{
					ReturnCode: respCode,
					Message:    message,
				},
			}, errors.New("failed to validate IP configs request")
		}
		ipConfigsResp, err := defaultHandler(ctx, req)
		// If the pod is not v2, return the response from the handler
		if !req.SecondaryInterfacesExist {
			return ipConfigsResp, err
		}
		// If the pod is v2, get the infra IP configs from the handler first and then add the SWIFTv2 IP config
		defer func() {
			// Release the default IP config if there is an error
			if err != nil {
				_, err = failureHandler(ctx, req)
				if err != nil {
					logger.Errorf("failed to release default IP config : %v", err)
				}
			}
		}()
		if err != nil {
			return ipConfigsResp, err
		}
		SWIFTv2PodIPInfos, err := k.getIPConfig(ctx, podInfo)
		if err != nil {
			return &cns.IPConfigsResponse{
				Response: cns.Response{
					ReturnCode: types.FailedToAllocateIPConfig,
					Message:    fmt.Sprintf("AllocateIPConfig failed: %v, IP config request is %v", err, req),
				},
				PodIPInfo: []cns.PodIpInfo{},
			}, errors.Wrapf(err, "failed to get SWIFTv2 IP config : %v", req)
		}
		ipConfigsResp.PodIPInfo = append(ipConfigsResp.PodIPInfo, SWIFTv2PodIPInfos...)
		// Set routes for the pod
		for i := range ipConfigsResp.PodIPInfo {
			ipInfo := &ipConfigsResp.PodIPInfo[i]
			err = k.setRoutes(ipInfo)
			if err != nil {
				return &cns.IPConfigsResponse{
					Response: cns.Response{
						ReturnCode: types.FailedToAllocateIPConfig,
						Message:    fmt.Sprintf("AllocateIPConfig failed: %v, IP config request is %v", err, req),
					},
					PodIPInfo: []cns.PodIpInfo{},
				}, errors.Wrapf(err, "failed to set routes for pod %s", podInfo.Name())
			}
		}
		return ipConfigsResp, nil
	}
}

// validateIPConfigsRequest validates if pod is multitenant by checking the pod labels, used in SWIFT V2 AKS scenario.
// nolint
func (k *K8sSWIFTv2Middleware) validateIPConfigsRequest(ctx context.Context, req *cns.IPConfigsRequest) (podInfo cns.PodInfo, respCode types.ResponseCode, message string) {
	// Retrieve the pod from the cluster
	podInfo, err := cns.UnmarshalPodInfo(req.OrchestratorContext)
	if err != nil {
		errBuf := errors.Wrapf(err, "failed to unmarshalling pod info from ipconfigs request %+v", req)
		return nil, types.UnexpectedError, errBuf.Error()
	}
	logger.Printf("[SWIFTv2Middleware] validate ipconfigs request for pod %s", podInfo.Name())
	podNamespacedName := k8stypes.NamespacedName{Namespace: podInfo.Namespace(), Name: podInfo.Name()}
	pod := v1.Pod{}
	if err := k.Cli.Get(ctx, podNamespacedName, &pod); err != nil {
		errBuf := errors.Wrapf(err, "failed to get pod %+v", podNamespacedName)
		return nil, types.UnexpectedError, errBuf.Error()
	}

	// check the pod labels for Swift V2, set the request's SecondaryInterfaceSet flag to true and check if its MTPNC CRD is ready
	if _, ok := pod.Labels[configuration.LabelPodSwiftV2]; ok {
		req.SecondaryInterfacesExist = true
		// Check if the MTPNC CRD exists for the pod, if not, return error
		mtpnc := v1alpha1.MultitenantPodNetworkConfig{}
		mtpncNamespacedName := k8stypes.NamespacedName{Namespace: podInfo.Namespace(), Name: podInfo.Name()}
		if err := k.Cli.Get(ctx, mtpncNamespacedName, &mtpnc); err != nil {
			return nil, types.UnexpectedError, fmt.Errorf("failed to get pod's mtpnc from cache : %w", err).Error()
		}
		// Check if the MTPNC CRD is ready. If one of the fields is empty, return error
		if !mtpnc.IsReady() {
			return nil, types.UnexpectedError, errMTPNCNotReady.Error()
		}
		interfaceInfos := mtpnc.Status.InterfaceInfos
		for index, interfaceInfo := range interfaceInfos {
			if interfaceInfo.NICType == string(cns.BackendNIC) {
				if interfaceInfo.MacAddress == "" || interfaceInfo.NCID == "" || mtpnc.Status.NCID == "" {
					return nil, types.UnexpectedError, errMTPNCNotReady.Error()
				}
				req.BackendInterfaceExist = true
				req.BackendInterfaceMacAddresses[index] = interfaceInfo.MacAddress
			}
		}
	}
	logger.Printf("[SWIFTv2Middleware] pod %s has secondary interface : %v", podInfo.Name(), req.SecondaryInterfacesExist)
	logger.Printf("[SWIFTv2Middleware] pod %s has backend interface : %v", podInfo.Name(), req.BackendInterfaceExist)
	// retrieve podinfo from orchestrator context
	return podInfo, types.Success, ""
}

// getIPConfig returns the pod's SWIFT V2 IP configuration.
func (k *K8sSWIFTv2Middleware) getIPConfig(ctx context.Context, podInfo cns.PodInfo) ([]cns.PodIpInfo, error) {
	// Check if the MTPNC CRD exists for the pod, if not, return error
	mtpnc := v1alpha1.MultitenantPodNetworkConfig{}
	mtpncNamespacedName := k8stypes.NamespacedName{Namespace: podInfo.Namespace(), Name: podInfo.Name()}
	if err := k.Cli.Get(ctx, mtpncNamespacedName, &mtpnc); err != nil {
		return nil, errors.Wrapf(err, "failed to get pod's mtpnc from cache")
	}

	// Check if the MTPNC CRD is ready. If one of the fields is empty, return error
	if !mtpnc.IsReady() {
		return nil, errMTPNCNotReady
	}
	logger.Printf("[SWIFTv2Middleware] mtpnc for pod %s is : %+v", podInfo.Name(), mtpnc)

	var podIPInfos []cns.PodIpInfo

	if len(mtpnc.Status.InterfaceInfos) == 0 {
		// Use fields from mtpnc.Status if InterfaceInfos is empty
		ip, prefixSize, err := utils.ParseIPAndPrefix(mtpnc.Status.PrimaryIP)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse mtpnc primary IP and prefix")
		}
		if prefixSize != prefixLength {
			return nil, errors.Wrapf(errInvalidMTPNCPrefixLength, "mtpnc primaryIP prefix length is %d", prefixSize)
		}

		podIPInfos = append(podIPInfos, cns.PodIpInfo{
			PodIPConfig: cns.IPSubnet{
				IPAddress:    ip,
				PrefixLength: uint8(prefixSize),
			},
			MacAddress:        mtpnc.Status.MacAddress,
			NICType:           cns.DelegatedVMNIC,
			SkipDefaultRoutes: false,
			// InterfaceName is empty for DelegatedVMNIC
		})
	} else {
		// Use InterfaceInfos if not empty
		podIPInfos = make([]cns.PodIpInfo, len(mtpnc.Status.InterfaceInfos))
		for i, interfaceInfo := range mtpnc.Status.InterfaceInfos {
			// Parse MTPNC primaryIP to get the IP address and prefix length
			ip, prefixSize, err := utils.ParseIPAndPrefix(interfaceInfo.PrimaryIP)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse mtpnc primary IP and prefix")
			}
			if prefixSize != prefixLength {
				return nil, errors.Wrapf(errInvalidMTPNCPrefixLength, "mtpnc primaryIP prefix length is %d", prefixSize)
			}

			var nicType cns.NICType
			switch {
			case interfaceInfo.DeviceType == v1alpha1.DeviceTypeVnetNIC && !interfaceInfo.AccelnetEnabled:
				nicType = cns.DelegatedVMNIC
			case interfaceInfo.DeviceType == v1alpha1.DeviceTypeVnetNIC && interfaceInfo.AccelnetEnabled:
				nicType = cns.NodeNetworkInterfaceAccelnetFrontendNIC
			case interfaceInfo.DeviceType == v1alpha1.DeviceTypeInfiniBandNIC:
				nicType = cns.NodeNetworkInterfaceBackendNIC
			default:
				nicType = cns.DelegatedVMNIC
			}

			podIPInfos[i] = cns.PodIpInfo{
				PodIPConfig: cns.IPSubnet{
					IPAddress:    ip,
					PrefixLength: uint8(prefixSize),
				},
				MacAddress:        interfaceInfo.MacAddress,
				NICType:           nicType,
				SkipDefaultRoutes: false,
			}
		}
	}

	return podIPInfos, nil
}
