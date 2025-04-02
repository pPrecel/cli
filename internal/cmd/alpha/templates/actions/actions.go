package actions

import (
	"github.com/kyma-project/cli.v3/internal/clierror"
	"github.com/kyma-project/cli.v3/internal/cmdcommon"
	"gopkg.in/yaml.v3"
)

func parseActionConfig[T any](in cmdcommon.CommandConfig, out *T) clierror.Error {
	if in == nil {
		return clierror.New("empty config object")
	}

	configBytes, err := yaml.Marshal(in)
	if err != nil {
		return clierror.Wrap(err, clierror.New("failed to marshal config"))
	}

	err = yaml.Unmarshal(configBytes, out)
	if err != nil {
		return clierror.Wrap(err, clierror.New("failed to unmarshal config"))
	}

	return nil
}
