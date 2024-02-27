// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package cns

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-container-networking/cns/common"
	"github.com/Azure/azure-container-networking/cns/logger"
	acn "github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/keyvault"
	"github.com/Azure/azure-container-networking/log"
	localtls "github.com/Azure/azure-container-networking/server/tls"
	"github.com/Azure/azure-container-networking/store"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/pkg/errors"
)

const (
	// Default CNS server URL.
	defaultAPIServerURL  = "tcp://localhost:10090"
	defaultAPIServerPort = "10090"
	genericData          = "com.microsoft.azure.network.generic"
	LocalListener        = "localListener"
	NodeListener         = "nodeListener"
)

type cnsListener struct {
	Listener     acn.Listener
	ListenerType string
}

// Service defines Container Networking Service.
type Service struct {
	*common.Service
	EndpointType string
	Listeners    []*cnsListener
}

// NewService creates a new Service object.
func NewService(name, version, channelMode string, store store.KeyValueStore) (*Service, error) {
	service, err := common.NewService(name, version, channelMode, store)
	if err != nil {
		return nil, err
	}

	return &Service{
		Service: service,
	}, nil
}

// AddListeners is to add two listeners(nodeListener and localListener) for connections on the given address
func (service *Service) AddListeners(config *common.ServiceConfig) error {

	// Fetch and parse the API server URL.
	var nodeURL *url.URL
	cnsURL, _ := service.GetOption(acn.OptCnsURL).(string)
	if cnsURL == "" {
		// get VM primary interface's private IP
		// primaryInterfaceIP, err := service.GetPrimaryInterfaceIP()
		// if err != nil {
		// 	logger.Errorf("[Azure CNS] Failed to get primary interface IP, err:%v", err)
		// 	return err
		// }
		nodeURL, _ = url.Parse(fmt.Sprintf("tcp://%s:%s", "127.0.0.1", defaultAPIServerPort))
	} else {
		// use the URL that customer provides
		nodeURL, _ = url.Parse(cnsURL)
	}

	cnsListener := cnsListener{}

	if config.ChannelMode != CRD {
		// construct url
		nodeListener, err := acn.NewListener(nodeURL)
		if err != nil {
			return errors.Wrap(err, "Failed to construct url for node listener")
		}

		// only use TLS connection for DNC/CNS listener:
		if config.TLSSettings.TLSPort != "" {
			// listener.URL.Host will always be hostname:port, passed in to CNS via CNS command
			// else it will default to localhost
			// extract hostname and override tls port.
			hostParts := strings.Split(nodeListener.URL.Host, ":")
			tlsAddress := net.JoinHostPort(hostParts[0], config.TLSSettings.TLSPort)

			// Start the listener and HTTP and HTTPS server.
			tlsConfig, err := getTLSConfig(config.TLSSettings, config.ErrChan) //nolint
			if err != nil {
				log.Printf("Failed to compose Tls Configuration with error: %+v", err)
				return errors.Wrap(err, "could not get tls config")
			}

			if err := nodeListener.StartTLS(config.ErrChan, tlsConfig, tlsAddress); err != nil {
				return errors.Wrap(err, "could not start tls")
			}
		}
		config.Listeners = append(config.Listeners, nodeListener)

		cnsListener.Listener = *nodeListener
		cnsListener.ListenerType = NodeListener

		service.Listeners = append(service.Listeners, &cnsListener)
	}

	// bind on localhost ip for CNI listener
	localURL, _ := url.Parse(defaultAPIServerURL)
	localListener, err := acn.NewListener(localURL)
	if err != nil {
		return errors.Wrap(err, "Failed to construct url for local listener")
	}
	cnsListener.Listener = *localListener
	cnsListener.ListenerType = NodeListener
	service.Listeners = append(service.Listeners, &cnsListener)

	logger.Printf("HTTP listeners will be started later after CNS state has been reconciled")

	return nil
}

// Initialize initializes the service and starts the listener.
func (service *Service) Initialize(config *common.ServiceConfig) error {
	log.Debugf("[Azure CNS] Going to initialize a service with config: %+v", config)

	// Initialize the base service.
	if err := service.Service.Initialize(config); err != nil {
		return errors.Wrap(err, "failed to initialize")
	}

	// Initialize listeners.
	if len(config.Listeners) == 0 {
		if err := service.AddListeners(config); err != nil {
			return errors.Wrap(err, "failed to initialize listener")
		}
	}

	log.Debugf("[Azure CNS] Successfully initialized a service with config: %+v", config)
	return nil
}

func getTLSConfig(tlsSettings localtls.TlsSettings, errChan chan<- error) (*tls.Config, error) {
	if tlsSettings.TLSCertificatePath != "" {
		return getTLSConfigFromFile(tlsSettings)
	}

	if tlsSettings.KeyVaultURL != "" {
		return getTLSConfigFromKeyVault(tlsSettings, errChan)
	}

	return nil, errors.Errorf("invalid tls settings: %+v", tlsSettings)
}

func getTLSConfigFromFile(tlsSettings localtls.TlsSettings) (*tls.Config, error) {
	tlsCertRetriever, err := localtls.GetTLSCertificateRetriever(tlsSettings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get certificate retriever")
	}

	leafCertificate, err := tlsCertRetriever.GetCertificate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get certificate")
	}

	if leafCertificate == nil {
		return nil, errors.New("certificate retrieval returned empty")
	}

	privateKey, err := tlsCertRetriever.GetPrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get certificate private key")
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{leafCertificate.Raw},
		PrivateKey:  privateKey,
		Leaf:        leafCertificate,
	}

	tlsConfig := &tls.Config{
		MaxVersion: tls.VersionTLS13,
		MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{
			tlsCert,
		},
	}

	return tlsConfig, nil
}

func getTLSConfigFromKeyVault(tlsSettings localtls.TlsSettings, errChan chan<- error) (*tls.Config, error) {
	credOpts := azidentity.ManagedIdentityCredentialOptions{ID: azidentity.ResourceID(tlsSettings.MSIResourceID)}
	cred, err := azidentity.NewManagedIdentityCredential(&credOpts)
	if err != nil {
		return nil, errors.Wrap(err, "could not create managed identity credential")
	}

	kvs, err := keyvault.NewShim(tlsSettings.KeyVaultURL, cred)
	if err != nil {
		return nil, errors.Wrap(err, "could not create new keyvault shim")
	}

	ctx := context.TODO()

	cr, err := keyvault.NewCertRefresher(ctx, kvs, logger.Log, tlsSettings.KeyVaultCertificateName)
	if err != nil {
		return nil, errors.Wrap(err, "could not create new cert refresher")
	}

	go func() {
		errChan <- cr.Refresh(ctx, tlsSettings.KeyVaultCertificateRefreshInterval)
	}()

	tlsConfig := tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return cr.GetCertificate(), nil
		},
	}

	return &tlsConfig, nil
}

func (service *Service) StartListener(config *common.ServiceConfig) error {
	log.Debugf("[Azure CNS] Going to start listeners: %+v", config)

	// Initialize listeners.
	for _, listener := range service.Listeners { //nolint
		if listener != nil {
			log.Debugf("[Azure CNS] Starting listener: %+v", config)
			// Start the listener.
			// continue to listen on the normal endpoint for http traffic, this will be supported
			// for sometime until partners migrate fully to https
			if err := listener.Listener.Start(config.ErrChan); err != nil {
				return errors.Wrap(err, "Failed to create a listener socket and starts the HTTP server")
			}
		} else {
			err := errors.Errorf("the listener is not initialized, config %+v", config)
			return errors.Wrap(err, "Failed to start a listener")
		}
	}

	return nil
}

// Uninitialize cleans up the plugin.
func (service *Service) Uninitialize() {
	for _, listener := range service.Listeners { //nolint
		listener.Listener.Stop()
	}
	service.Service.Uninitialize()
}

// ParseOptions returns generic options from a libnetwork request.
func (service *Service) ParseOptions(options OptionMap) OptionMap {
	opt, _ := options[genericData].(OptionMap)
	return opt
}

// SendErrorResponse sends and logs an error response.
func (service *Service) SendErrorResponse(w http.ResponseWriter, errMsg error) {
	resp := errorResponse{errMsg.Error()}
	err := acn.Encode(w, &resp)
	log.Errorf("[%s] %+v %s.", service.Name, &resp, err.Error())
}
