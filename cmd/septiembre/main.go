// Command septiembre is the agent-first CLI for the Septiembre cloud platform.
//
// It emits JSON to stdout by default and JSON error envelopes to stderr.
// All exit codes are defined in internal/output/exit.go.
package main

import (
	"fmt"
	"os"

	"github.com/septiembre-ai/septiembre-cli/internal/cli"
	"github.com/septiembre-ai/septiembre-cli/internal/output"
)

func main() {
	root := cli.NewRootCmd()
	if err := root.Execute(); err != nil {
		// Write error as JSON to stderr so agents can parse it.
		fmt.Fprintf(os.Stderr, `{"error":%q,"code":"general_error"}`+"\n", err.Error())
		os.Exit(output.ExitGeneral)
	}
}
