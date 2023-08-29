package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

type StopCmd struct{}

// NewStopCmd returns a new start command
func NewStopCmd() *cobra.Command {
	cmd := &StopCmd{}
	cobraCmd := &cobra.Command{
		Use:           "stop",
		Short:         "Stops the container",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return cobraCmd
}

func (cmd *StopCmd) Run(ctx context.Context) error {
	// check if we already have built the image
	if !isContainerRunning() {
		return fmt.Errorf("container is not running")
	}

	// check if we already have built the image
	_, err := os.Stat(ContainerPID)
	if err != nil {
		return fmt.Errorf("container is not started")
	}

	out, err := os.ReadFile(ContainerPID)
	if err != nil {
		return fmt.Errorf("read pid file: %w", err)
	}

	parsedPid, err := strconv.Atoi(string(out))
	if err != nil {
		return fmt.Errorf("parse process id: %w", err)
	}

	_ = syscall.Kill(parsedPid, syscall.SIGTERM)
	time.Sleep(500)
	_ = syscall.Kill(parsedPid, syscall.SIGKILL)

	err = os.Remove(ContainerPID)
	if err != nil {
		return fmt.Errorf("remove pid file: %w", err)
	}

	err = os.Remove(ContainerConfigOutput)
	if err != nil {
		return fmt.Errorf("remove config file: %w", err)
	}

	return nil
}
