package function

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/client"
	"github.com/kyma-incubator/hydroform/function/pkg/docker"
	"github.com/kyma-incubator/hydroform/function/pkg/docker/runtimes"
	"github.com/kyma-incubator/hydroform/function/pkg/workspace"
	"github.com/kyma-project/cli/internal/cli"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
		Use:   "function",
		Short: "Runs Functions locally.",
		Long:  `Use this command to run a Function in Docker from local sources`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Filename, "filename", "f", "", `Full path to the config file.`)
	cmd.Flags().StringVarP(&o.Dir, "sourceDir", "d", "", `Full path to the folder with the source code.`)
	cmd.Flags().StringVar(&o.ContainerName, "containerName", "", `The name of the created container.`)
	cmd.Flags().BoolVar(&o.Detach, "detach", false, `Change this flag to "true" if you don't want to follow the container logs after running the Function.`)
	cmd.Flags().StringVarP(&o.FuncPort, "port", "p", "8080", `The port on which the container will be exposed.`)

	//cmd.Flags().BoolVarP(&o.Debug, "debug", "d", false, `Change this flat to true if you want to expose port 9229 for remote debugging.`)

	return cmd
}

func (c *command) Run() error {
	if c.opts.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := c.opts.setDefaults(); err != nil {
		return err
	}

	cfg, err := workspaceConfig(c.opts.Filename)
	if err != nil {
		return err
	}

	// git functions are not supported yet
	if cfg.Source.Type == workspace.SourceTypeGit {
		return errors.New("The git source type of functions is not supported yet")
	}

	if c.opts.ContainerName == "" {
		c.opts.ContainerName = cfg.Name
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "white trying to interact with docker")
	}

	return c.runContainer(ctx, client, cfg)
}

func workspaceConfig(path string) (workspace.Cfg, error) {
	file, err := os.Open(path)
	if err != nil {
		return workspace.Cfg{}, err
	}
	var cfg workspace.Cfg
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return workspace.Cfg{}, errors.Wrap(err, "while trying to decode the configuration file")
	}

	if cfg.Runtime != "nodejs12" &&
		cfg.Runtime != "nodejs10" &&
		cfg.Runtime != "python38" {
		return workspace.Cfg{}, fmt.Errorf("unsupported runtime: %s", cfg.Runtime)
	}

	return cfg, nil
}

func (c *command) runContainer(ctx context.Context, client *client.Client, cfg workspace.Cfg) error {
	c.newStep(fmt.Sprintf("Running container: %s", c.opts.ContainerName))
	ports := map[string]string{
		c.opts.FuncPort: runtimes.ServerPort,
	}
	if c.opts.Debug {
		debugPort := runtimes.RuntimeDebugPort(cfg.Runtime)
		ports[debugPort] = debugPort
	}
	id, err := docker.RunContainer(ctx, client, docker.RunOpts{
		Ports: ports,
		Envs: append(
			runtimes.ContainerEnvs(cfg.Runtime, c.opts.Debug),
			c.parseEnvs(cfg.Env)...,
		),
		ContainerName: c.opts.ContainerName,
		// TODO: ImageName:     runtimes.ImageName(cfg.Runtime),
		WorkingDir: c.opts.Dir,
	})
	if err != nil {
		return errors.Wrap(err, "white trying to run container")
	}

	c.successStepf(fmt.Sprintf("Runned container: %s", c.opts.ContainerName))

	if !c.opts.Detach {
		fmt.Println("Logs from the container:")
		followCtx := context.Background()
		c.Finalizers.Add(docker.Stop(followCtx, client, id, func(i ...interface{}) { fmt.Print(i...) }))
		return docker.FollowRun(followCtx, client, id, func(i ...interface{}) { fmt.Print(i...) })
	}
	return nil
}

func (c *command) parseEnvs(envVars []workspace.EnvVar) []string {
	var envs []string
	for _, env := range envVars {
		envs = append(envs, fmt.Sprintf("%s:%s", env.Name, env.Value))
	}
	return envs
}

func (c *command) newStep(message string) {
	if c.opts.Verbose {
		logrus.Debug(message)
	} else {
		c.NewStep(message)
	}
}

func (c *command) successStepf(message string) {
	if c.opts.Verbose {
		logrus.Debug(message)
	} else if c.CurrentStep != nil {
		c.CurrentStep.Successf(message)
	}
}
