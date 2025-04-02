package alpha

import (
	"github.com/kyma-project/cli.v3/internal/cmd/alpha/app"
	"github.com/kyma-project/cli.v3/internal/cmd/alpha/hana"
	"github.com/kyma-project/cli.v3/internal/cmd/alpha/kubeconfig"
	"github.com/kyma-project/cli.v3/internal/cmd/alpha/module"
	"github.com/kyma-project/cli.v3/internal/cmd/alpha/provision"
	"github.com/kyma-project/cli.v3/internal/cmd/alpha/referenceinstance"
	"github.com/kyma-project/cli.v3/internal/cmdcommon"
	"github.com/kyma-project/cli.v3/internal/extensions"
	"github.com/kyma-project/cli.v3/internal/extensions/actions"
	extensions_types "github.com/kyma-project/cli.v3/internal/extensions/types"
	"github.com/spf13/cobra"
)

func NewAlphaCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "alpha <command> [flags]",
		Short:                 "Groups command prototypes for which the API may still change",
		Long:                  `A set of alpha prototypes that may still change. Use them in automation at your own risk.`,
		DisableFlagsInUseLine: true,
	}

	kymaConfig := cmdcommon.NewKymaConfig()

	cmd.AddCommand(app.NewAppCMD(kymaConfig))
	cmd.AddCommand(hana.NewHanaCMD(kymaConfig))
	cmd.AddCommand(module.NewModuleCMD(kymaConfig))
	cmd.AddCommand(provision.NewProvisionCMD())
	cmd.AddCommand(referenceinstance.NewReferenceInstanceCMD(kymaConfig))
	cmd.AddCommand(kubeconfig.NewKubeconfigCMD(kymaConfig))

	builder := extensions.NewBuilder()
	cmds := builder.Build(kymaConfig, extensions_types.ActionsMap{
		"function_init":         actions.NewFunctionInit,
		"registry_config":       actions.NewRegistryConfig,
		"registry_image-import": actions.NewRegistryImageImport,
		"resource_create":       actions.NewResourceCreate,
		"resource_get":          actions.NewResourceGet,
		"resource_delete":       actions.NewResourceDelete,
		"resource_explain":      actions.NewResourceExplain,
	})

	builder.DisplayWarnings(cmd.ErrOrStderr())

	cmd.AddCommand(cmds...)

	return cmd
}
