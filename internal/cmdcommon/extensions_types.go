package cmdcommon

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kyma-project/cli.v3/internal/cmd/alpha/templates/parameters"
	"github.com/spf13/cobra"
)

const (
	ExtensionCMLabelKey   = "kyma-cli/extension"
	ExtensionCMLabelValue = "commands"
	ExtensionCMDataKey    = "kyma-commands.yaml"
)

// map of allowed action commands in format ID: FUNC
type UsesMap map[string]func(*KymaConfig, CommandConfig) *cobra.Command

// TODO: add config validation

type CommandConfig = map[string]interface{}

type ConfigmapCommandExtension struct {
	ConfigMapName      string
	ConfigMapNamespace string
	Extension          CommandExtension
}

type CommandMetadata struct {
	// name of the command group
	Name string `yaml:"name"`
	// short description of the command group
	Description string `yaml:"description"`
	// long description of the command group
	DescriptionLong string `yaml:"descriptionLong"`
}

func (m *CommandMetadata) Validate() error {
	if m.Name == "" {
		return errors.New("empty name")
	}

	return nil
}

type CommandArgs struct {
	// type of the argument and config field
	// TODO: support many args by adding new type, like `stringArray`
	Type parameters.ConfigFieldType `yaml:"type"`
	// mark if args are required to run command
	Optional bool `yaml:"optional"`
	// path to the config fild that will be updated with value from args
	ConfigPath string `yaml:"configPath"`
}

func (a *CommandArgs) Validate() error {
	var err error
	if !slices.Contains(parameters.ValidTypes, a.Type) {
		errors.Join(err, fmt.Errorf("unknown type '%s'", a.Type))
	}

	if a.ConfigPath == "" {
		errors.Join(err, errors.New("empty ConfigPath"))
	}

	return err
}

type CommandFlag struct {
	// type of the flag and config field
	Type parameters.ConfigFieldType `yaml:"type"`
	// name of the flag
	Name string `yaml:"name"`
	// description of the flag
	Description string `yaml:"description"`
	// optional shorthand of the flag
	Shorthand string `yaml:"shorthand"`
	// path to the config fild that will be updated with value from the flag
	ConfigPath string `yaml:"configPath"`
	// default value for the flag
	DefaultValue string `yaml:"default"`
	// mark if flag is required
	Required bool `yaml:"required"`
}

func (f *CommandFlag) Validate() error {
	var err error
	if !slices.Contains(parameters.ValidTypes, f.Type) {
		errors.Join(err, fmt.Errorf("unknown type '%s'", f.Type))
	}

	if f.ConfigPath == "" {
		errors.Join(err, errors.New("empty ConfigPath"))
	}

	// TODO: what about DefaultValue?

	return err
}

type CommandExtension struct {
	// metadata (name, descriptions) for the command
	Metadata CommandMetadata `yaml:"metadata"`
	// id of the functionality that cli will run when user use this command
	Uses string `yaml:"uses"`
	// flags used to set specific fields in config
	Flags []CommandFlag `yaml:"flags"`
	// args used to set specific fields in config
	Args *CommandArgs `yaml:"args"`
	// additional config pass to the command
	Config CommandConfig `yaml:"config"`
	// list of sub commands
	SubCommands []CommandExtension `yaml:"subCommands"`
}

func (e *CommandExtension) Validate(availableActions UsesMap) error {
	return e.validateWithPath(".", availableActions)
}

func (e *CommandExtension) validateWithPath(path string, availableActions UsesMap) error {
	var err error
	if _, ok := availableActions[e.Uses]; e.Uses != "" && !ok {
		err = errors.Join(err, fmt.Errorf("wrong %suses: unsupported value '%s'", path, e.Uses))
	}

	// run sub-validations
	if metaErr := e.Metadata.Validate(); metaErr != nil {
		err = errors.Join(err, fmt.Errorf("wrong %smetadata: %s", path, metaErr.Error()))
	}

	if e.Args != nil {
		if argsErr := e.Args.Validate(); argsErr != nil {
			err = errors.Join(err, fmt.Errorf("wrong %sargs: %s", argsErr.Error()))
		}
	}

	for i := range e.Flags {
		if flagErr := e.Flags[i].Validate(); flagErr != nil {
			err = errors.Join(err, fmt.Errorf("wrong %sflags: %s", path, flagErr))
		}
	}

	for i := range e.SubCommands {
		subCmdErr := e.SubCommands[i].validateWithPath(fmt.Sprintf("%ssubCommands[%d].", path, i), availableActions)
		if subCmdErr != nil {
			err = errors.Join(err, subCmdErr)
		}
	}

	return err
}
