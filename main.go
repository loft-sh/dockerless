package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/loft-sh/dockerless/cmd"
)

func main() {
	// build the root command
	rootCmd := cmd.NewRootCmd()

	// execute command
	err := rootCmd.Execute()
	if err != nil {
		//nolint:all
		if execExitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(execExitErr.ExitCode())
		}

		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
