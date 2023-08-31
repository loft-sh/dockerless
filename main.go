package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/loft-sh/dockerless/cmd"
)

func main() {
	// build the root command
	rootCmd := cmd.NewRootCmd()

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	// execute command
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		execExitErr := &exec.ExitError{}
		if errors.As(err, &execExitErr) {
			stop()
			os.Exit(execExitErr.ExitCode())
		}

		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		stop()
		os.Exit(1)
	}

	stop()
}
