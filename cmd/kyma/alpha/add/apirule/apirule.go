package apirule

import (
	"io/ioutil"
	"os"

	"github.com/kyma-incubator/hydroform/function/pkg/workspace"
	"github.com/kyma-project/cli/internal/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type command struct {
	opts *Options
	cli.Command
}

//NewCmd creates a new init command
func NewCmd(o *Options) *cobra.Command {
	c := command{
		opts:    o,
		Command: cli.Command{Options: o.Options},
	}
	cmd := &cobra.Command{
		Use:   "api-rule",
		Short: "TODO",
		Long:  `TODO`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Filename, "filename", "f", "", ``)

	cmd.Flags().StringVarP(&o.Name, "name", "n", "", `ApiRule name.`)
	cmd.Flags().StringVar(&o.Gateway, "gateway", "", ``)
	cmd.Flags().StringVar(&o.Host, "host", "", ``)
	cmd.Flags().Int64Var(&o.Port, "port", 0, ``)
	cmd.Flags().StringVar(&o.Path, "path", "", ``)
	cmd.Flags().StringVar(&o.Handler, "handler", "", ``)
	cmd.Flags().StringArrayVar(&o.Methods, "methods", []string{"GET"}, ``)
	cmd.Flags().StringArrayVar(&o.JwksUrls, "jwks-urls", []string{}, ``)
	cmd.Flags().StringArrayVar(&o.RequiredScope, "required-scope", []string{}, ``)
	cmd.Flags().StringArrayVar(&o.TrustedIssuers, "trusted-issuers", []string{}, ``)

	return cmd
}

func (c *command) Run() error {
	s := c.NewStep("Getting accual config file")

	if err := c.opts.setDefaults(); err != nil {
		s.Failure()
		return err
	}

	file, err := os.Open(c.opts.Filename)
	if err != nil {
		s.Failure()
		return err
	}
	defer file.Close()

	// Load project configuration
	var configuration workspace.Cfg
	if err := yaml.NewDecoder(file).Decode(&configuration); err != nil {
		s.Failure()
		return errors.Wrap(err, "Could not decode the configuration file")
	}

	var asList []workspace.AccessStrategie
	if c.opts.Handler != "allow" && c.opts.Handler != "" {
		asList = append(asList, workspace.AccessStrategie{
			Config: workspace.AccessStrategieConfig{
				JwksUrls:       c.opts.JwksUrls,
				TrustedIssuers: c.opts.TrustedIssuers,
				RequiredScope:  c.opts.RequiredScope,
			},
			Handler: c.opts.Handler,
		})
	}

	configuration.ApiRules = append(configuration.ApiRules, workspace.ApiRule{
		Name:    c.opts.Name,
		Gateway: c.opts.Gateway,
		Service: workspace.Service{
			Host: c.opts.Host,
			Port: c.opts.Port,
		},
		Rules: []workspace.Rule{
			{
				Path:             c.opts.Path,
				Methods:          c.opts.Methods,
				AccessStrategies: asList,
			},
		},
	})

	rawCfg, err := yaml.Marshal(&configuration)
	if err != nil {
		s.Failure()
		return err
	}

	err = ioutil.WriteFile(c.opts.Filename, rawCfg, 0644)
	if err != nil {
		s.Failure()
		return err
	}

	s.Successf("ApiRule generated in %s", c.opts.Filename)
	return nil
}
