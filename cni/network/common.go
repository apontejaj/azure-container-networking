package network

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/cni/log"
	"github.com/Azure/azure-container-networking/telemetry"
	"github.com/containernetworking/cni/pkg/skel"
	cniTypes "github.com/containernetworking/cni/pkg/types"
	"go.uber.org/zap"
)

// send error report to hostnetagent if CNI encounters any error.
func ReportPluginError(reportManager *telemetry.ReportManager, tb *telemetry.TelemetryBuffer, err error) {
	log.Logger.Error("Report plugin error")
	reflect.ValueOf(reportManager.Report).Elem().FieldByName("ErrorMessage").SetString(err.Error())

	if err := reportManager.SendReport(tb); err != nil {
		log.Logger.Error("SendReport failed", zap.Error(err))
	}
}

func validateConfig(jsonBytes []byte) error {
	var conf struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(jsonBytes, &conf); err != nil {
		return fmt.Errorf("error reading network config: %s", err)
	}
	if conf.Name == "" {
		return fmt.Errorf("missing network name")
	}
	return nil
}

func getCmdArgsFromEnv() (string, *skel.CmdArgs, error) {
	log.Logger.Info("Going to read from stdin")
	stdinData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", nil, fmt.Errorf("error reading from stdin: %v", err)
	}

	cmdArgs := &skel.CmdArgs{
		ContainerID: os.Getenv("CNI_CONTAINERID"),
		Netns:       os.Getenv("CNI_NETNS"),
		IfName:      os.Getenv("CNI_IFNAME"),
		Args:        os.Getenv("CNI_ARGS"),
		Path:        os.Getenv("CNI_PATH"),
		StdinData:   stdinData,
	}

	cmd := os.Getenv("CNI_COMMAND")
	return cmd, cmdArgs, nil
}

func HandleIfCniUpdate(update func(*skel.CmdArgs) error) (bool, error) {
	isupdate := true

	if os.Getenv("CNI_COMMAND") != cni.CmdUpdate {
		return false, nil
	}

	log.Logger.Info("CNI UPDATE received")

	_, cmdArgs, err := getCmdArgsFromEnv()
	if err != nil {
		log.Logger.Error("Received error while retrieving cmds from environment", zap.Error(err))
		return isupdate, err
	}

	log.Logger.Info("Retrieved command args for update", zap.Any("args", cmdArgs))
	err = validateConfig(cmdArgs.StdinData)
	if err != nil {
		log.Logger.Error("Failed to handle CNI UPDATE", zap.Error(err))
		return isupdate, err
	}

	err = update(cmdArgs)
	if err != nil {
		log.Logger.Error("Failed to handle CNI UPDATE", zap.Error(err))
		return isupdate, err
	}

	return isupdate, nil
}

func PrintCNIError(msg string) {
	log.Logger.Error(msg)
	cniErr := &cniTypes.Error{
		Code: cniTypes.ErrTryAgainLater,
		Msg:  msg,
	}
	cniErr.Print()
}
