package apirule

import (
	"fmt"
	"os"

	"github.com/kyma-incubator/hydroform/function/pkg/workspace"
	"github.com/kyma-project/cli/internal/cli"
)

type Options struct {
	*cli.Options

	Filename string

	Name           string
	Gateway        string
	Host           string
	Port           int64
	Path           string
	Handler        string
	Methods        []string
	JwksUrls       []string
	TrustedIssuers []string
	RequiredScope  []string
}

func NewOptions(o *cli.Options) *Options {
	return &Options{Options: o}
}

func (o *Options) setDefaults() error {
	if o.Filename == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		o.Filename = fmt.Sprintf("%s/%s", wd, workspace.CfgFilename)
	}

	return nil
}
