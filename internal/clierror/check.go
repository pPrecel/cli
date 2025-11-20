package clierror

import (
	"os"

	"github.com/kyma-project/cli.v3/internal/out"
)

// Check prints error and executes os.Exit(1) if the error is not nil
func Check(err Error) {
	if err != nil {
		out.Errln(withHelpHint(err).String())
		os.Exit(1)
	}
}

func withHelpHint(err Error) Error {
	clierr := err.(*clierror)
	clierr.hints = append(
		clierr.hints,
		"use --help to see available commands and examples",
	)

	return clierr
}
