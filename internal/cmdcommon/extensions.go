package cmdcommon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/kyma-project/cli.v3/internal/clierror"
	"github.com/kyma-project/cli.v3/internal/cmd/alpha/templates/parameters"
	"github.com/kyma-project/cli.v3/internal/cmdcommon/flags"
	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KymaExtensionsConfig struct {
	kymaConfig      *KymaConfig
	extensionsError error
}

func newExtensionsConfig(kymaConfig *KymaConfig) *KymaExtensionsConfig {
	return &KymaExtensionsConfig{
		kymaConfig: kymaConfig,
	}
}

func AddExtensionsFlags(cmd *cobra.Command) {
	// these flags are not operational. it's only to print the help description, and the help cobra with validation
	_ = cmd.PersistentFlags().Bool("skip-extensions", false, "Skip fetching extensions from the cluster")
	_ = cmd.PersistentFlags().Bool("show-extensions-error", false, "Prints a possible error when fetching extensions fails")
}

func (kec *KymaExtensionsConfig) DisplayExtensionsErrors(warningWriter io.Writer) {
	if isSubRootCommandUsed("help", "completion", "version") {
		// skip if one of restricted flags is used
		return
	}

	if kec.extensionsError != nil && getBoolFlagValue("--show-extensions-error") {
		// print error as warning if expected and continue
		fmt.Fprintf(warningWriter, "Extensions Warning:\n%s\n\n", kec.extensionsError.Error())
	} else if kec.extensionsError != nil {
		fmt.Fprintf(warningWriter, "Extensions Warning:\nfailed to fetch all extensions from the cluster. Use the '--show-extensions-error' flag to see more details.\n\n")
	}
}

// build extensions based on extensions configmaps from a cluster
// any errors can be displayed by using the DisplayExtensionsErrors func
func (kec *KymaExtensionsConfig) BuildExtensions(availableActions UsesMap) []*cobra.Command {
	if getBoolFlagValue("--skip-extensions") {
		// skip extensions fetching
		return nil
	}

	var configmapExtensions []ConfigmapCommandExtension
	configmapExtensions, kec.extensionsError = loadCommandExtensionsFromCluster(kec.kymaConfig.Ctx, kec.kymaConfig.KubeClientConfig)
	if kec.extensionsError != nil {
		// set extensionsError and stop
		return nil
	}

	extensions := []*cobra.Command{}
	for _, cmExt := range configmapExtensions {
		err := cmExt.Extension.Validate(availableActions)
		if err != nil {
			kec.extensionsError = errors.Join(
				kec.extensionsError,
				pkgerrors.Wrapf(err, "failed to validate extension from configmap '%s/%s'", cmExt.ConfigMapName, cmExt.ConfigMapNamespace),
			)
			continue
		}

		command, err := kec.buildExtensionCommand(cmExt.Extension, availableActions)
		if err != nil {
			kec.extensionsError = errors.Join(
				kec.extensionsError,
				pkgerrors.Wrapf(err, "failed to build extension from configmap '%s/%s'", cmExt.ConfigMapName, cmExt.ConfigMapNamespace),
			)
			continue
		}

		extensions = append(extensions, command)
	}

	return extensions
}

func (kec *KymaExtensionsConfig) buildExtensionCommand(extension CommandExtension, availableActions UsesMap) (*cobra.Command, error) {
	var buildError error

	cmd := availableActions[extension.Uses](kec.kymaConfig, extension.Config)
	cmd.Use = extension.Metadata.Name
	cmd.Short = extension.Metadata.Description
	cmd.Long = extension.Metadata.DescriptionLong

	// set flags
	values := []parameters.Value{}
	requiredFlags := []string{}
	for _, flag := range extension.Flags {
		value, err := kec.newTypedValue(flag.Type, flag.ConfigPath, flag.DefaultValue)
		if err != nil {
			buildError = errors.Join(buildError,
				fmt.Errorf("failed to build flag for '%s' command: %s", extension.Metadata.Name, err.Error()))
		}

		cmdFlag := cmd.Flags().VarPF(value, flag.Name, flag.Shorthand, flag.Description)
		if flag.Type == parameters.BoolCustomType {
			// set default value for bool flag used without value (for example "--flag" instead of "--flag value")
			cmdFlag.NoOptDefVal = "true"
		}
		if flag.Required {
			requiredFlags = append(requiredFlags, flag.Name)
		}

		values = append(values, value)
	}

	// set args
	if extension.Args != nil {
		value := parameters.NewTyped(extension.Args.Type, extension.Args.ConfigPath)
		values = append(values, value)
		cmd.Args = func(_ *cobra.Command, args []string) error {
			if extension.Args.Optional {
				return setOptionalArg(value, args)
			}
			return setRequiredArg(value, args)
		}
	}

	// build sub-commands
	for _, subCommand := range extension.SubCommands {
		subCmd, subErr := kec.buildExtensionCommand(subCommand, availableActions)
		if subErr != nil {
			buildError = errors.Join(buildError,
				fmt.Errorf("failed to build sub-command '%s': %s", subCommand.Metadata.Name, subErr.Error()))
		}

		cmd.AddCommand(subCmd)
	}

	cmd.PreRun = func(_ *cobra.Command, _ []string) {
		// check required flags
		clierror.Check(flags.Validate(cmd.Flags(),
			flags.MarkRequired(requiredFlags...),
		))
		// set parameters from flag
		clierror.Check(parameters.Set(extension.Config, values))
	}

	return cmd, buildError
}

func (kec *KymaExtensionsConfig) newTypedValue(valueType parameters.ConfigFieldType, configPath, defaultValue string) (parameters.Value, error) {
	var err error
	value := parameters.NewTyped(valueType, configPath)
	if defaultValue != "" {
		setErr := value.SetValue(defaultValue)
		if setErr != nil {
			err = pkgerrors.Wrapf(setErr, "failed to set default value '%s'", defaultValue)
		}
	}

	return value, err
}

func loadCommandExtensionsFromCluster(ctx context.Context, clientConfig *KubeClientConfig) ([]ConfigmapCommandExtension, error) {
	var cms, cmsError = listCommandExtenionConfigMaps(ctx, clientConfig)
	if cmsError != nil {
		return nil, cmsError
	}

	extensions := []ConfigmapCommandExtension{}
	var parseErrors error
	for _, cm := range cms.Items {
		commandExtension, err := parseRequiredField[CommandExtension](cm.Data, ExtensionCMDataKey)
		if err != nil {
			// if the parse failed add an error to the errors list to take another extension
			// corrupted extension should not stop parsing the rest of the extensions
			parseErrors = errors.Join(
				parseErrors,
				pkgerrors.Wrapf(err, "failed to parse configmap '%s/%s'", cm.GetNamespace(), cm.GetName()),
			)
			continue
		}

		if slices.ContainsFunc(extensions, func(e ConfigmapCommandExtension) bool {
			return e.Extension.Metadata.Name == commandExtension.Metadata.Name
		}) {
			parseErrors = errors.Join(
				parseErrors,
				fmt.Errorf("failed to validate configmap '%s/%s': extension with rootCommand.name='%s' already exists",
					cm.GetNamespace(), cm.GetName(), commandExtension.Metadata.Name),
			)
			continue
		}

		extensions = append(extensions, ConfigmapCommandExtension{
			ConfigMapName:      cm.GetName(),
			ConfigMapNamespace: cm.GetNamespace(),
			Extension:          *commandExtension,
		})
	}

	return extensions, parseErrors
}

func listCommandExtenionConfigMaps(ctx context.Context, clientConfig *KubeClientConfig) (*v1.ConfigMapList, error) {
	client, clientErr := clientConfig.GetKubeClient()
	if clientErr != nil {
		return nil, clientErr
	}

	labelSelector := fmt.Sprintf("%s==%s", ExtensionCMLabelKey, ExtensionCMLabelValue)
	cms, err := client.Static().CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "failed to load ConfigMaps from cluster with label %s", labelSelector)
	}

	return cms, nil
}

func parseRequiredField[T any](cmData map[string]string, cmKey string) (*T, error) {
	dataBytes, ok := cmData[cmKey]
	if !ok {
		return nil, fmt.Errorf("missing .data.%s field", cmKey)
	}

	var data T
	err := yaml.Unmarshal([]byte(dataBytes), &data)
	return &data, err
}

func setOptionalArg(value parameters.Value, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("accepts at most one argument, received %d", len(args))
	}

	if len(args) != 0 {
		return value.Set(args[0])
	}

	return nil
}

func setRequiredArg(value parameters.Value, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("requires exactly one argument, received %d", len(args))
	}

	return value.Set(args[0])
}

// search os.Args manually to find if user pass given flag and return its value
func getBoolFlagValue(flag string) bool {
	for i, arg := range os.Args {
		//example: --show-extensions-error true
		if arg == flag && len(os.Args) > i+1 {

			value, err := strconv.ParseBool(os.Args[i+1])
			if err == nil {
				return value
			}
		}

		// example: --show-extensions-error or --show-extensions-error=true
		if strings.HasPrefix(arg, flag) && !strings.Contains(arg, "false") {
			return true
		}
	}

	return false
}

// checks if one of given args is on the 2 possition of os.Args (first sub-command)
func isSubRootCommandUsed(args ...string) bool {
	for _, arg := range args {
		if slices.Contains(os.Args, arg) {
			return true
		}
	}

	return false
}
