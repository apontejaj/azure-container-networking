package k8s

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Azure/azure-container-networking/test/e2e/types"
	k8s "github.com/Azure/azure-container-networking/test/integration"
	"github.com/Azure/azure-container-networking/test/internal/retry"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	retryAttempts         = 15
	defaultTimeoutSeconds = 120
)

var (
	defaultRetrier = retry.Retrier{Attempts: 60, Delay: time.Second}
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

func (p *PortForward) Run(values *types.JobValues) error {
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
		fmt.Printf("attempting port forward to a pod with label %s, in namespace %s...", p.LabelSelector, p.Namespace)
		handle, err := p.pf.Forward(pctx, p.Namespace, p.LabelSelector, lport, rport)
		if err != nil {
			return fmt.Errorf("could not start port forward: %w", err)
		}
		p.portForwardHandle = handle
		return nil
	}
	fmt.Printf("streamHandle: %v\n", p.portForwardHandle.Url())

	if err = defaultRetrier.Do(portForwardCtx, portForwardFn); err != nil {
		return fmt.Errorf("could not start port forward within %ds: %v", defaultTimeoutSeconds, err)
	}

	return nil
}

func (p *PortForward) ExpectError() bool {
	return false
}

func (p *PortForward) SaveParametersToJob() bool {
	return true
}

func (p *PortForward) Prevalidate(values *types.JobValues) error {
	return nil
}

func (p *PortForward) Postvalidate(values *types.JobValues) error {
	p.portForwardHandle.Stop()
	return nil
}
