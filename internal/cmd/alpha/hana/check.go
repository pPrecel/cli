package hana

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/cli.v3/internal/clierror"
	"github.com/kyma-project/cli.v3/internal/cmdcommon"
	"github.com/kyma-project/cli.v3/internal/kube/btp"
	"github.com/spf13/cobra"
)

type hanaCheckConfig struct {
	*cmdcommon.KymaConfig
	cmdcommon.KubeClientConfig

	name      string
	namespace string
	timeout   time.Duration
}

func NewHanaCheckCMD(kymaConfig *cmdcommon.KymaConfig) *cobra.Command {
	config := hanaCheckConfig{
		KymaConfig:       kymaConfig,
		KubeClientConfig: cmdcommon.KubeClientConfig{},
	}

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check if the Hana instance is provisioned.",
		Long:  "Use this command to check if the Hana instance is provisioned on the SAP Kyma platform.",
		PreRun: func(_ *cobra.Command, _ []string) {
			clierror.Check(config.KubeClientConfig.Complete())
		},
		Run: func(_ *cobra.Command, _ []string) {
			clierror.Check(runCheck(&config))
		},
	}

	config.KubeClientConfig.AddFlag(cmd)

	cmd.Flags().StringVar(&config.name, "name", "", "Name of Hana instance.")
	cmd.Flags().StringVar(&config.namespace, "namespace", "default", "Namespace for Hana instance.")

	_ = cmd.MarkFlagRequired("name")

	return cmd
}

var (
	checkCommands = []func(config *hanaCheckConfig) clierror.Error{
		checkHanaInstance,
		checkHanaBinding,
		checkHanaBindingURL,
	}
)

func runCheck(config *hanaCheckConfig) clierror.Error {
	fmt.Printf("Checking Hana (%s/%s).\n", config.namespace, config.name)

	for _, command := range checkCommands {
		err := command(config)
		if err != nil {
			fmt.Println("Hana is not fully ready.")
			return clierror.New("failed to get resource data", "Make sure that Hana was provisioned.")
		}
	}
	fmt.Println("Hana is fully ready.")
	return nil
}

func checkHanaInstance(config *hanaCheckConfig) clierror.Error {
	instance, err := config.KubeClient.Btp().GetServiceInstance(config.Ctx, config.namespace, config.name)
	if err != nil {
		return clierror.New(err.Error())
	}

	return isResourceReady(instance.Status, "Hana instance", config.namespace, config.name)
}

func checkHanaBinding(config *hanaCheckConfig) clierror.Error {
	binding, err := config.KubeClient.Btp().GetServiceBinding(config.Ctx, config.namespace, config.name)
	if err != nil {
		return clierror.New(err.Error())
	}

	return isResourceReady(binding.Status, "Hana binding", config.namespace, config.name)
}

func checkHanaBindingURL(config *hanaCheckConfig) clierror.Error {
	urlName := hanaBindingURLName(config.name)
	binding, err := config.KubeClient.Btp().GetServiceBinding(config.Ctx, config.namespace, urlName)
	if err != nil {
		return clierror.New(err.Error())
	}

	return isResourceReady(binding.Status, "Hana URL binding", config.namespace, urlName)
}

func isResourceReady(status btp.CommonStatus, printedName, namespace, name string) clierror.Error {
	ready := status.IsReady()
	if !ready {
		fmt.Printf("%s is not ready (%s/%s).\n", printedName, namespace, name)
		errMsg := fmt.Sprintf("%s is not ready", strings.ToLower(printedName[:1])+printedName[1:])
		return clierror.New(errMsg, "Wait for provisioning of Hana resources.", "Check if Hana resources started without errors.")
	}
	fmt.Printf("%s is ready (%s/%s).\n", printedName, namespace, name)
	return nil
}
