// Command septiembre is the agent-first CLI for the Septiembre cloud platform.
//
// It emits JSON to stdout by default and JSON error envelopes to stderr.
// All exit codes are defined in internal/output/exit.go.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/septiembre-ai/septiembre-cli/internal/cli"
	"github.com/septiembre-ai/septiembre-cli/internal/output"
)

func main() {
	root := cli.NewRootCmd()
	if err := root.Execute(); err != nil {
		// Commands that set a specific exit code return *cli.ExitError and have
		// already written the error envelope to stderr via the Renderer.
		var ee *cli.ExitError
		if errors.As(err, &ee) {
			os.Exit(ee.Code)
		}
		// Fallback for unhandled errors (should not normally occur).
		fmt.Fprintf(os.Stderr, `{"error":%q,"code":"general_error"}`+"\n", err.Error())
		os.Exit(output.ExitGeneral)
	}
}
