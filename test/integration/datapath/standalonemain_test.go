//go:build connection

package connection

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	namespace                     = "default"
	podPrefix                     = "swiftpod-"
	ncNodeLabelSelector           = "agentpool=podpool"
	nodeLabelKey                  = "dncnode"
	nodeAddressType               = "InternalIP"
	hostDaemonsetVolumeMountPath  = "/host"
	cnsStateFilePath              = hostDaemonsetVolumeMountPath + "/var/lib/azure-network/azure-cns.json"
	logDir                        = "logs/"
	cnsLogFileName                = "azure-cns.log"
	cniLogFileName                = "azure-vnet.log"
	goldpingerLogFileName         = "goldpinger.log"
	dncPodDescribeFileName        = "dnc_pod_describe.txt"
	cnsPodDescribeFileName        = "cns_pod_describe.txt"
	cniManagerPodDescribeFileName = "cni_pod_describe.txt"
	goldpingerPodDescribeFileName = "goldpinger_pod_describe.txt"
	cnsStateFileName              = "azure-cns.json"
	defaultTimeoutSeconds         = 120
	defaultRetryDelaySeconds      = 1
	maxRetryDelaySeconds          = 10
	deleteWaitTimeSeconds         = 20
	deleteNCWaitTimeInSeconds     = 60
	goldpingerRetryCount          = 24
	goldpingerDelayTimeSeconds    = 5
)

var (
	restConfig *rest.Config
	clientset  *kubernetes.Clientset
	k8sShim    *k8s.Shim
	testConfig = &TestConfig{}

	// todo: these should not need to be globals for the sake of clean up.
	// t.Cleanup can be called with multiple clean up funcs instead of an uber one.
	nodeNameToNodeInfo     map[string]*nodeInfo
	invalidSubnetTokenNCID string
)

type TestConfig struct {
	CNIManagerImage             string `env:"CNI_MANAGER_IMAGE"`
	CNSImage                    string `env:"CNS_IMAGE"`
	CNIManagerDaemonsetYamlPath string `env:"CNI_MANAGER_DAEMONSET_YAML_PATH"`
	CNSDaemonsetYamlPath        string `env:"CNS_DAEMONSET_YAML_PATH"`
	CNSConfigmapYamlPath        string `env:"CNS_CONFIGMAP_YAML_PATH"`
	GoldpingerPodYamlPath       string `env:"GOLDPINGER_POD_YAML_PATH"`
	HostDaemonsetYamlPath       string `env:"HOST_DAEMONSET_YAML_PATH"`
	PartitionKey                string `env:"PARTITION_KEY"`
	EnableAZRKey                string `env:"ENABLE_AZR_KEY"`
	DesiredNCsPerNode           int    `env:"DESIRED_NCS_PER_NODE"`
	InfraVnetGuid               string `env:"INFRA_VNET_GUID"`
	CustomerVnetGuid            string `env:"CUSTOMER_VNET_GUID"`
	CustomerSubnetName          string `env:"CUSTOMER_SUBNET_NAME"`
	DelegationToken             string `env:"DELEGATION_TOKEN"`
}

func TestMain(m *testing.M) {
	var err error
	if restConfig, err = config.GetConfig(); err != nil {
		logrus.Fatalf("could not get k8s rest config: %v", err)
	}

	logrus.Infof("k8s rest config for apiserver %s", restConfig.Host)

	if clientset, err = kubernetes.NewForConfig(restConfig); err != nil {
		logrus.Fatalf("could not get k8s clientset: %v", err)
	}

	k8sShim = k8s.NewShim(clientset)

	LoadEnvironment(testConfig)

	osExitCode := m.Run()

	os.Exit(osExitCode)
}

func LoadEnvironment(obj interface{}) {
	val := reflect.ValueOf(obj).Elem()
	typ := reflect.TypeOf(obj).Elem()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldTyp := typ.Field(i)

		env := fieldTyp.Tag.Get("env")
		envVal := os.Getenv(env)

		if envVal == "" {
			panic(fmt.Sprintf("required environment variable %q is missing", env))
		}

		switch fieldVal.Kind() {
		case reflect.Int:
			intVal, err := strconv.Atoi(envVal)
			if err != nil {
				panic(fmt.Sprintf("environment variable %q must be an integer", env))
			}
			fieldVal.SetInt(int64(intVal))
		case reflect.String:
			fieldVal.SetString(envVal)
		}
	}
}
