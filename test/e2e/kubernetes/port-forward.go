package kubernetes

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
	retryAttempts = 10
	retryDelay    = 5 * time.Second
)

var (
	defaultRetrier = retry.Retrier{Attempts: retryAttempts, Delay: retryDelay}
)

type PortForward struct {
	Namespace          string
	LabelSelector      string
	LocalPort          string
	RemotePort         string
	KubeConfigFilePath string
}

func (p *PortForward) Run(values *types.JobValues) error {
	config, err := clientcmd.BuildConfigFromFlags("", p.KubeConfigFilePath)
	if err != nil {
		fmt.Println("Error building kubeconfig: ", err)
		return err
	}
	lport, _ := strconv.Atoi(p.LocalPort)
	rport, _ := strconv.Atoi(p.RemotePort)

	pf, err := k8s.NewPortForwarder(config, nil, k8s.PortForwardingOpts{
		Namespace:     p.Namespace,
		LabelSelector: p.LabelSelector,
		LocalPort:     lport,
		DestPort:      rport,
	})

	if err != nil {
		return fmt.Errorf("could not create port forwarder: %w", err)
	}
	pctx := context.Background()

	portForwardCtx, cancel := context.WithTimeout(pctx, (retryAttempts+1)*retryDelay)
	defer cancel()

	portForwardFn := func() error {
		fmt.Println("attempting port forward to a pod with label %s, in namespace %s...", p.LabelSelector, p.Namespace)
		if err = pf.Forward(portForwardCtx); err != nil {
			return fmt.Errorf("could not start port forward: %w", err)
		}
		return nil
	}

	if err = defaultRetrier.Do(portForwardCtx, portForwardFn); err != nil {
		return fmt.Errorf("could not start port forward within %d: %v", (retryAttempts+1)*retryDelay, err)
	}

	return nil
}

func (p *PortForward) ExpectError() bool {
	return false
}

func (c *PortForward) SaveParametersToJob() bool {
	return true
}

func (c *PortForward) Prevalidate(values *types.JobValues) error {
	return nil
}
