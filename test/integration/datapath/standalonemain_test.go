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

	deleteWaitTimeSeconds     = 20
	deleteNCWaitTimeInSeconds = 60
)

var (
	restConfig *rest.Config
	clientset  *kubernetes.Clientset
	testConfig = &TestConfig{}

	// todo: these should not need to be globals for the sake of clean up.
	// t.Cleanup can be called with multiple clean up funcs instead of an uber one.
	nodeNameToNodeInfo     map[string]*nodeInfo
	invalidSubnetTokenNCID string
)

type TestConfig struct {
	GoldpingerPodYamlPath string `env:"GOLDPINGER_POD_YAML_PATH"`
	DesiredNCsPerNode     int    `env:"DESIRED_NCS_PER_NODE"`
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
