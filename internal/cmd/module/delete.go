package module

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/kyma-project/cli.v3/internal/clierror"
	"github.com/kyma-project/cli.v3/internal/cmdcommon"
	"github.com/kyma-project/cli.v3/internal/cmdcommon/prompt"
	"github.com/kyma-project/cli.v3/internal/flags"
	"github.com/kyma-project/cli.v3/internal/kube"
	"github.com/kyma-project/cli.v3/internal/modules"
	"github.com/kyma-project/cli.v3/internal/modules/repo"
	"github.com/kyma-project/cli.v3/internal/out"
	"github.com/spf13/cobra"
)

type deleteConfig struct {
	*cmdcommon.KymaConfig
	autoApprove bool
	community   bool

	modules     []string
	modulePaths []string
}

func newDeleteCMD(kymaConfig *cmdcommon.KymaConfig) *cobra.Command {
	cfg := deleteConfig{
		KymaConfig: kymaConfig,
	}

	cmd := &cobra.Command{
		Use:     "delete <module/s> [flags]",
		Short:   "Deletes a modules",
		Long:    "Use this command to delete a module.",
		Aliases: []string{"del"},
		Args:    cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, _ []string) {
			clierror.Check(flags.Validate(cmd.Flags(),
				flags.MarkUnsupported("community", "the --community flag is no longer supported - specify community module to delete using argument"),
			))
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg.complete(args)
			clierror.Check(runDelete(&cfg))
		},
	}

	cmd.Flags().BoolVar(&cfg.autoApprove, "auto-approve", false, "Automatically approves module removal")

	cmd.Flags().BoolVar(&cfg.community, "community", false, "Delete the community module (if set, the operation targets a community module instead of a core module)")
	_ = cmd.Flags().MarkHidden("community")

	return cmd
}

func (c *deleteConfig) complete(args []string) {
	for _, arg := range args {
		if strings.Contains(arg, "/") {
			// arg is module location in format <namespace>/<module-template-name>
			c.modulePaths = append(c.modulePaths, arg)
		} else {
			// arg is module name
			c.modules = append(c.modules, arg)
		}
	}
}

func runDelete(cfg *deleteConfig) clierror.Error {
	client, clierr := cfg.GetKubeClientWithClierr()
	if clierr != nil {
		return clierr
	}

	// delete core modules
	for i, module := range cfg.modules {
		clierr := disableModule(cfg.Ctx, client, module, cfg.autoApprove)
		if clierr != nil {
			return clierror.WrapE(clierr, clierror.New(fmt.Sprintf("failed to delete module '%s'", module)))
		}

		if i < len(cfg.modules)-1 || len(cfg.modulePaths) > 0 {
			// print empty line between multiple module deletions
			out.Msgln("")
		}
	}

	// delete community modules
	for i, modulePath := range cfg.modulePaths {
		clierr := uninstallCommunityModule(cfg.Ctx, client, modulePath, cfg.autoApprove)
		if clierr != nil {
			return clierror.WrapE(clierr, clierror.New(fmt.Sprintf("failed to delete community module '%s'", modulePath)))
		}

		if i < len(cfg.modulePaths)-1 {
			// print empty line between multiple module deletions
			out.Msgln("")
		}
	}

	return nil
}

func uninstallCommunityModule(ctx context.Context, client kube.Client, modulePath string, autoApprove bool) clierror.Error {
	repo := repo.NewModuleTemplatesRepo(client)
	namespace, moduleTemplateName, err := validateOrigin(modulePath)
	if err != nil {
		return clierror.Wrap(err, clierror.New("failed to identify the community module"))
	}

	communityModuleTemplate, err := modules.FindCommunityModuleTemplate(ctx, namespace, moduleTemplateName, repo)
	if err != nil {
		return clierror.Wrap(err, clierror.New("failed to retrieve the module '%s/%s'", namespace, moduleTemplateName))
	}

	if !autoApprove {
		runningResources, clierr := modules.GetRunningResourcesOfCommunityModule(ctx, repo, *communityModuleTemplate)
		if clierr != nil {
			return clierr
		}
		if len(runningResources) > 0 {
			confirmationPrompt := prompt.NewBool(prepareCommunityPromptMessage(runningResources), false)
			confirmation, err := confirmationPrompt.Prompt()
			if err != nil {
				return clierror.Wrap(err, clierror.New("failed to prompt for user input", "if error repeats, consider running the command with --auto-approve flag"))
			}

			if !confirmation {
				return nil
			}
		}
	}

	return modules.Uninstall(ctx, repo, communityModuleTemplate)
}

func disableModule(ctx context.Context, client kube.Client, module string, autoApprove bool) clierror.Error {
	if !autoApprove {
		confirmationPrompt := prompt.NewBool(prepareCorePromptMessage(module), false)
		confirmation, err := confirmationPrompt.Prompt()
		if err != nil {
			return clierror.Wrap(err, clierror.New("failed to prompt for user input", "if error repeats, consider running the command with --auto-approve flag"))
		}

		if !confirmation {
			return nil
		}
	}

	return modules.Disable(ctx, client, module)
}

func prepareCommunityPromptMessage(resourcesNames []string) string {
	var buf bytes.Buffer

	fmt.Fprint(&buf, "There are currently associated resources related to this module still running on the cluster:\n")
	for _, name := range resourcesNames {
		fmt.Fprintf(&buf, "  - %s\n", name)
	}
	fmt.Fprint(&buf, "\nDeleting the module may affect these resources.\n")
	fmt.Fprint(&buf, "Are you sure you want to proceed with the deletion?")

	return buf.String()
}

func prepareCorePromptMessage(moduleName string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Are you sure you want to delete module %s?\n", moduleName)
	fmt.Fprintf(&buf, "Before you delete the %s module, make sure the module resources are no longer needed. This action also permanently removes the namespaces, service instances, and service bindings created by the module.\n", moduleName)
	fmt.Fprintf(&buf, "Are you sure you want to continue?")

	return buf.String()
}
