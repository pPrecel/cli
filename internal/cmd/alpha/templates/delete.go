package templates

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/kyma-project/cli.v3/internal/clierror"
	"github.com/kyma-project/cli.v3/internal/cmd/alpha/templates/parameters"
	"github.com/kyma-project/cli.v3/internal/cmd/alpha/templates/types"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type DeleteOptions struct {
	types.DeleteCommand
	ResourceInfo types.ResourceInfo
}

func BuildDeleteCommand(clientGetter KubeClientGetter, options *DeleteOptions) *cobra.Command {
	return buildDeleteCommand(os.Stdout, clientGetter, options)
}

func buildDeleteCommand(out io.Writer, clientGetter KubeClientGetter, options *DeleteOptions) *cobra.Command {
	extraValues := []parameters.Value{}
	cmd := &cobra.Command{
		Use:   "delete",
		Short: options.Description,
		Long:  options.DescriptionLong,
		Run: func(cmd *cobra.Command, args []string) {
			clierror.Check(deleteResource(&deleteArgs{
				out:           out,
				ctx:           cmd.Context(),
				deleteOptions: options,
				clientGetter:  clientGetter,
				extraValues:   extraValues,
			}))
		},
	}

	// define resource name as required args[0]
	nameArgValue := parameters.NewTyped(resourceNameFlag.Type, resourceNameFlag.Path, resourceNameFlag.DefaultValue)
	cmd.Args = AssignRequiredNameArg(nameArgValue)
	extraValues = append(extraValues, nameArgValue)

	// define --namespace/-n flag only is resource is namespace scoped
	if options.ResourceInfo.Scope == types.NamespaceScope {
		namespaceFlagValue := parameters.NewTyped(resourceNamespaceFlag.Type, resourceNamespaceFlag.Path, resourceNamespaceFlag.DefaultValue)
		cmd.Flags().VarP(namespaceFlagValue, resourceNamespaceFlag.Name, resourceNamespaceFlag.Shorthand, resourceNamespaceFlag.Description)
		extraValues = append(extraValues, namespaceFlagValue)
	}

	return cmd
}

type deleteArgs struct {
	out           io.Writer
	ctx           context.Context
	deleteOptions *DeleteOptions
	clientGetter  KubeClientGetter
	extraValues   []parameters.Value
}

func deleteResource(args *deleteArgs) clierror.Error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   args.deleteOptions.ResourceInfo.Group,
		Version: args.deleteOptions.ResourceInfo.Version,
		Kind:    args.deleteOptions.ResourceInfo.Kind,
	})

	client, clierr := args.clientGetter.GetKubeClientWithClierr()
	if clierr != nil {
		return clierr
	}

	clierr = parameters.Set(u, args.extraValues)
	if clierr != nil {
		return clierr
	}

	err := client.RootlessDynamic().Remove(args.ctx, u)
	if err != nil {
		return clierror.Wrap(err, clierror.New("failed to delete resource"))
	}

	fmt.Fprintf(args.out, "resource %s deleted\n", getResourceName(args.deleteOptions.ResourceInfo.Scope, u))

	return nil
}
