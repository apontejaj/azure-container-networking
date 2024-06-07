// Copyright Microsoft. All rights reserved.
package configuration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/common"
	"github.com/pkg/errors"
)

type SWIFTV2Mode string

const (
	// EnvCNSConfig is the CNS_CONFIGURATION_PATH env var key
	EnvCNSConfig      = "CNS_CONFIGURATION_PATH"
	defaultConfigName = "cns_config.json"
	// Service Fabric SWIFTV2 mode
	SFSWIFTV2 SWIFTV2Mode = "SFSWIFTV2"
	// K8s SWIFTV2 mode
	K8sSWIFTV2 SWIFTV2Mode = "K8sSWIFTV2"
)

type CNSConfig struct {
	AZRSettings                 AZRSettings
	AsyncPodDeletePath          string
	CNIConflistFilepath         string
	CNIConflistScenario         string
	ChannelMode                 string
	EnableAsyncPodDelete        bool
	EnableCNIConflistGeneration bool
	EnableIPAMv2                bool
	EnablePprof                 bool
	EnableStateMigration        bool
	EnableSubnetScarcity        bool
	EnableSwiftV2               bool
	InitializeFromCNI           bool
	KeyVaultSettings            KeyVaultSettings
	MSISettings                 MSISettings
	ManageEndpointState         bool
	ManagedSettings             ManagedSettings
	MellanoxMonitorIntervalSecs int
	MetricsBindAddress          string
	ProgramSNATIPTables         bool
	SWIFTV2Mode                 SWIFTV2Mode
	SyncHostNCTimeoutMs         int
	SyncHostNCVersionIntervalMs int
	TLSCertificatePath          string
	TLSEndpoint                 string
	TLSPort                     string
	TLSSubjectName              string
	TelemetrySettings           TelemetrySettings
	UseHTTPS                    bool
	WatchPods                   bool `json:"-"`
	WireserverIP                string
	GRPCSettings                GRPCSettings
}

type TelemetrySettings struct {
	// Flag to disable the telemetry.
	DisableAll bool
	// Flag to Disable sending trace.
	DisableTrace bool
	// Flag to Disable sending metric.
	DisableMetric bool
	// Flag to Disable sending events.
	DisableEvent bool
	// Configure how many bytes can be sent in one call to the data collector
	TelemetryBatchSizeBytes int
	// Configure the maximum delay before sending queued telemetry in milliseconds
	TelemetryBatchIntervalInSecs int
	// Heartbeat interval for sending heartbeat metric
	HeartBeatIntervalInMins int
	// Enable thread for getting metadata from wireserver
	DisableMetadataRefreshThread bool
	// Refresh interval in milliseconds for metadata thread
	RefreshIntervalInSecs int
	// Disable debug logging for telemetry messages
	DebugMode bool
	// Interval for sending snapshot events.
	SnapshotIntervalInMins int
	// AppInsightsInstrumentationKey allows the user to override the default appinsights ikey
	AppInsightsInstrumentationKey string
}

type ManagedSettings struct {
	PrivateEndpoint           string
	InfrastructureNetworkID   string
	NodeID                    string
	NodeSyncIntervalInSeconds int
}

type AZRSettings struct {
	PopulateHomeAzCacheRetryIntervalSecs int
}

type MSISettings struct {
	ResourceID string
}

type KeyVaultSettings struct {
	URL                  string
	CertificateName      string
	RefreshIntervalInHrs int
}

type GRPCSettings struct {
	Enable          bool
	ServerIPAddress string
	ServerPort      uint16
}

func getConfigFilePath(cmdPath string) (string, error) {
	// If config path is set from cmd line, return that.
	if strings.TrimSpace(cmdPath) != "" {
		return cmdPath, nil
	}
	// If config path is set from env, return that.
	if envPath := os.Getenv(EnvCNSConfig); strings.TrimSpace(envPath) != "" {
		return envPath, nil
	}
	// otherwise compose the default config path and return that.
	dir, err := common.GetExecutableDirectory()
	if err != nil {
		return "", errors.Wrap(err, "failed to discover exec dir for config")
	}
	defaultPath := filepath.Join(dir, defaultConfigName)
	return defaultPath, nil
}

// ReadConfig returns a CNS config from file or an error.
func ReadConfig(cmdLineConfigPath string) (*CNSConfig, error) {
	configpath, err := getConfigFilePath(cmdLineConfigPath)
	if err != nil {
		return nil, err
	}
	logger.Printf("[Configuration] Using config path: %s", configpath)
	return readConfigFromFile(configpath)
}

// readConfigFromFile attempts to read the file and unmarshal it in to a CNSConfig.
func readConfigFromFile(f string) (*CNSConfig, error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read config file %s", f)
	}
	var config CNSConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
	}
	return &config, nil
}

// set telmetry setting defaults
func setTelemetrySettingDefaults(telemetrySettings *TelemetrySettings) {
	if telemetrySettings.RefreshIntervalInSecs == 0 {
		// set the default refresh interval of metadata thread to 15 seconds
		telemetrySettings.RefreshIntervalInSecs = 15
	}

	if telemetrySettings.TelemetryBatchIntervalInSecs == 0 {
		// set the default AI telemetry batch interval to 30 seconds
		telemetrySettings.TelemetryBatchIntervalInSecs = 30
	}

	if telemetrySettings.TelemetryBatchSizeBytes == 0 {
		// set the default AI telemetry batch size to 32768 bytes
		telemetrySettings.TelemetryBatchSizeBytes = 32768
	}

	if telemetrySettings.HeartBeatIntervalInMins == 0 {
		// set the default Heartbeat interval to 30 minutes
		telemetrySettings.HeartBeatIntervalInMins = 30
	}

	if telemetrySettings.SnapshotIntervalInMins == 0 {
		telemetrySettings.SnapshotIntervalInMins = 60
	}
}

// set managed setting defaults
func setManagedSettingDefaults(managedSettings *ManagedSettings) {
	if managedSettings.NodeSyncIntervalInSeconds == 0 {
		managedSettings.NodeSyncIntervalInSeconds = 30
	}
}

func setAZRSettingsDefaults(azrSettings *AZRSettings) {
	if azrSettings.PopulateHomeAzCacheRetryIntervalSecs == 0 {
		// set the default PopulateHomeAzCache retry interval to 60 seconds
		azrSettings.PopulateHomeAzCacheRetryIntervalSecs = 60
	}
}

func setKeyVaultSettingsDefaults(kvs *KeyVaultSettings) {
	if kvs.RefreshIntervalInHrs == 0 {
		kvs.RefreshIntervalInHrs = 12 //nolint:gomnd // default times
	}
}

// SetCNSConfigDefaults set default values of CNS config if not specified
func SetCNSConfigDefaults(config *CNSConfig) {
	setTelemetrySettingDefaults(&config.TelemetrySettings)
	setManagedSettingDefaults(&config.ManagedSettings)
	setKeyVaultSettingsDefaults(&config.KeyVaultSettings)
	setAZRSettingsDefaults(&config.AZRSettings)

	if config.ChannelMode == "" {
		config.ChannelMode = cns.Direct
	}
	if config.MetricsBindAddress == "" {
		config.MetricsBindAddress = ":9090"
	}
	if config.SyncHostNCVersionIntervalMs == 0 {
		config.SyncHostNCVersionIntervalMs = 1000 //nolint:gomnd // default times
	}
	if config.SyncHostNCTimeoutMs == 0 {
		config.SyncHostNCTimeoutMs = 500 //nolint:gomnd // default times
	}
	if config.WireserverIP == "" {
		config.WireserverIP = "168.63.129.16"
	}
	if config.AsyncPodDeletePath == "" {
		config.AsyncPodDeletePath = "/var/run/azure-vnet/deleteIDs"
	}
	if config.GRPCSettings.ServerIPAddress == "" {
		config.GRPCSettings.ServerIPAddress = "localhost"
	}
	if config.GRPCSettings.ServerPort == 0 {
		config.GRPCSettings.ServerPort = 8080
	}
	config.GRPCSettings.Enable = true
	config.WatchPods = config.EnableIPAMv2 || config.EnableSwiftV2
}
