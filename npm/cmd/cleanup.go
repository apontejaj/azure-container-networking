package main

import (
	"fmt"
	"time"

	"github.com/Azure/azure-container-networking/common"
	npmconfig "github.com/Azure/azure-container-networking/npm/config"
	"github.com/Azure/azure-container-networking/npm/metrics"
	"github.com/Azure/azure-container-networking/npm/pkg/dataplane"
	"github.com/Azure/azure-container-networking/npm/pkg/dataplane/ipsets"
	"github.com/Azure/azure-container-networking/npm/pkg/models"
	"github.com/Azure/azure-container-networking/npm/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

// newCleanupNPMCmd returns the cleanup command, which deletes NPM state in the dataplane.
func newCleanupNPMCmd() *cobra.Command {
	// getTuplesCmd represents the getTuples command
	startNPMCmd := &cobra.Command{
		Use:   "start",
		Short: "Starts the Azure NPM process",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &npmconfig.Config{}
			err := viper.Unmarshal(config)
			if err != nil {
				return fmt.Errorf("failed to load config with error: %w", err)
			}

			flags := npmconfig.Flags{
				KubeConfigPath: viper.GetString(flagKubeConfigPath),
			}

			return cleanup(*config, flags)
		},
	}

	startNPMCmd.Flags().String(flagKubeConfigPath, flagDefaults[flagKubeConfigPath], "path to kubeconfig")

	return startNPMCmd
}

func cleanup(config npmconfig.Config, flags npmconfig.Flags) error {
	klog.Infof("loaded config: %+v", config)
	if util.IsWindowsDP() {
		config.Toggles.EnableV2NPM = true
		klog.Infof("NPM is running on Windows Dataplane. Enabling V2 NPM")
	} else {
		klog.Infof("NPM is running on Linux Dataplane")
	}
	klog.Infof("starting cleanup for NPM version %d with image %s", config.NPMVersion(), version)

	var err error

	err = initLogging()
	if err != nil {
		return err
	}

	// have to initialize metrics to prevent panic from modifying nil Prometheus metrics
	klog.Infof("initializing metrics")
	metrics.InitializeAll()

	// Create the kubernetes client
	var k8sConfig *rest.Config
	if flags.KubeConfigPath == "" {
		klog.Infof("loading in cluster kubeconfig")
		k8sConfig, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("failed to load in cluster config: %w", err)
		}
	} else {
		klog.Infof("loading kubeconfig from flag: %s", flags.KubeConfigPath)
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", flags.KubeConfigPath)
		if err != nil {
			return fmt.Errorf("failed to load kubeconfig [%s] with err config: %w", flags.KubeConfigPath, err)
		}
	}

	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		klog.Infof("clientset creation failed with error %v.", err)
		return fmt.Errorf("failed to generate clientset with cluster config: %w", err)
	}

	nodeName := models.GetNodeName()

	stopChannel := wait.NeverStop
	if config.Toggles.EnableV2NPM {
		// update the dataplane config
		npmV2DataplaneCfg.MaxBatchedACLsPerPod = config.MaxBatchedACLsPerPod

		npmV2DataplaneCfg.ApplyInBackground = config.Toggles.ApplyInBackground
		if config.ApplyMaxBatches > 0 {
			npmV2DataplaneCfg.ApplyMaxBatches = config.ApplyMaxBatches
		} else {
			npmV2DataplaneCfg.ApplyMaxBatches = npmconfig.DefaultConfig.ApplyMaxBatches
		}
		if config.ApplyIntervalInMilliseconds > 0 {
			npmV2DataplaneCfg.ApplyInterval = time.Duration(config.ApplyIntervalInMilliseconds * int(time.Millisecond))
		} else {
			npmV2DataplaneCfg.ApplyInterval = time.Duration(npmconfig.DefaultConfig.ApplyIntervalInMilliseconds * int(time.Millisecond))
		}

		if config.WindowsNetworkName == "" {
			npmV2DataplaneCfg.NetworkName = util.AzureNetworkName
		} else {
			npmV2DataplaneCfg.NetworkName = config.WindowsNetworkName
		}

		npmV2DataplaneCfg.PlaceAzureChainFirst = config.Toggles.PlaceAzureChainFirst
		if config.Toggles.ApplyIPSetsOnNeed {
			npmV2DataplaneCfg.IPSetMode = ipsets.ApplyOnNeed
		} else {
			npmV2DataplaneCfg.IPSetMode = ipsets.ApplyAllIPSets
		}

		var nodeIP string
		if util.IsWindowsDP() {
			nodeIP, err = util.NodeIP()
			if err != nil {
				metrics.SendErrorLogAndMetric(util.NpmCleanupID, "error: failed to get node IP while booting up: %v", err)
				return fmt.Errorf("failed to get node IP while booting up: %w", err)
			}
			klog.Infof("node IP is %s", nodeIP)
		}
		npmV2DataplaneCfg.NodeIP = nodeIP

		_, err := dataplane.NewDataPlane(nodeName, common.NewIOShim(), npmV2DataplaneCfg, stopChannel)
		if err != nil {
			metrics.SendErrorLogAndMetric(util.NpmCleanupID, "error: failed to create dataplane with error %v", err)
			return fmt.Errorf("failed to create dataplane with error %w", err)
		}
	}

	metrics.SendLog(util.NpmCleanupID, "finished cleanup", metrics.PrintLog)

	if err := labelNode(clientset, nodeName, util.RemovedLabelValue); err != nil {
		metrics.SendErrorLogAndMetric(util.NpmCleanupID, "error: failed to label node as NPM removed. err: %s", err.Error())
		return err
	}

	metrics.SendLog(util.NpmCleanupID, "labeled node as NPM removed", metrics.PrintLog)

	return nil
}
