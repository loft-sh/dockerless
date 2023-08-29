package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
)

type ExecCmd struct {
	User string
}

// NewExecCmd returns a new start command
func NewExecCmd() *cobra.Command {
	cmd := &ExecCmd{}
	cobraCmd := &cobra.Command{
		Use:           "exec",
		Short:         "Execs into the container",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.User, "user", "", "The user to execute the command as")
	return cobraCmd
}

func (cmd *ExecCmd) Run(ctx context.Context, args []string) error {
	// check if we already have built the image
	if !isContainerRunning() {
		return fmt.Errorf("container is not running")
	}

	// load container.json
	out, err := os.ReadFile(ContainerConfigOutput)
	if err != nil {
		return fmt.Errorf("read container config: %w", err)
	}

	// parse config file
	configFile := &v1.ConfigFile{}
	err = json.Unmarshal(out, configFile)
	if err != nil {
		return fmt.Errorf("parse container config: %w", err)
	}

	// override user
	if cmd.User != "" {
		configFile.Config.User = cmd.User
	}

	// get user info
	homeDir, credential, err := getUserInfo(configFile.Config.User)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// prepare the command to run
	if len(args) == 0 {
		return fmt.Errorf("no entrypoint specified")
	}

	// build the container env
	containerEnv, err := getContainerEnv(homeDir, configFile.Config.Env)
	if err != nil {
		return fmt.Errorf("get container env: %w", err)
	}

	// start command
	containerCmd := exec.Command(args[0], args[1:]...)
	containerCmd.SysProcAttr = &syscall.SysProcAttr{}
	containerCmd.SysProcAttr.Credential = credential
	containerCmd.Stdout = os.Stdout
	containerCmd.Stderr = os.Stderr
	containerCmd.Stdin = os.Stdin
	containerCmd.Dir = configFile.Config.WorkingDir
	containerCmd.Env = containerEnv
	return containerCmd.Run()
}
