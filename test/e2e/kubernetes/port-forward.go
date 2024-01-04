package k8s

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	k8s "github.com/Azure/azure-container-networking/test/integration"
	"github.com/Azure/azure-container-networking/test/internal/retry"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultTimeoutSeconds = 300
)

var (
	defaultRetrier = retry.Retrier{Attempts: 60, Delay: 5 * time.Second}
)

type PortForward struct {
	Namespace          string
	LabelSelector      string
	LocalPort          string
	RemotePort         string
	KubeConfigFilePath string

	// local properties
	pf                *k8s.PortForwarder
	portForwardHandle k8s.PortForwardStreamHandle
}

func (p *PortForward) Run() error {
	config, err := clientcmd.BuildConfigFromFlags("", p.KubeConfigFilePath)
	if err != nil {
		fmt.Println("Error building kubeconfig: ", err)
		return err
	}
	lport, _ := strconv.Atoi(p.LocalPort)
	rport, _ := strconv.Atoi(p.RemotePort)

	p.pf, err = k8s.NewPortForwarder(config)

	if err != nil {
		return fmt.Errorf("could not create port forwarder: %w", err)
	}
	pctx := context.Background()

	portForwardCtx, cancel := context.WithTimeout(pctx, defaultTimeoutSeconds*time.Second)
	defer cancel()

	portForwardFn := func() error {
		log.Printf("attempting port forward to a pod with label %s, in namespace %s...\n", p.LabelSelector, p.Namespace)
		handle, err := p.pf.Forward(pctx, p.Namespace, p.LabelSelector, lport, rport)
		if err != nil {
			return fmt.Errorf("could not start port forward: %w", err)
		}

		// verify port forward succeeded
		client := http.Client{
			Timeout: 2 * time.Second,
		}
		resp, err := client.Get(handle.Url()) //nolint
		if err != nil {
			log.Printf("port forward validation HTTP request to %s failed: %v\n", handle.Url(), err)
			handle.Stop()
			return fmt.Errorf("port forward validation HTTP request to %s failed: %w", handle.Url(), err)
		}
		defer resp.Body.Close()

		log.Printf("port forward validation HTTP request to %s succeeded, response: %s\n", handle.Url(), resp.Status)
		p.portForwardHandle = handle
		return nil
	}

	if err = defaultRetrier.Do(portForwardCtx, portForwardFn); err != nil {
		return fmt.Errorf("could not start port forward within %ds: %v", defaultTimeoutSeconds, err)
	}
	log.Printf("successfully port forwarded to %s\n", p.portForwardHandle.Url())
	return nil
}

func (p *PortForward) ExpectError() bool {
	return false
}

func (p *PortForward) SaveParametersToJob() bool {
	return true
}

func (p *PortForward) Prevalidate() error {
	return nil
}

func (p *PortForward) Postvalidate() error {
	p.portForwardHandle.Stop()
	return nil
}
