package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/spf13/cobra"
)

type RemoveCmd struct{}

// NewRemoveCmd returns a new start command
func NewRemoveCmd() *cobra.Command {
	cmd := &RemoveCmd{}
	cobraCmd := &cobra.Command{
		Use:           "remove",
		Short:         "Removes the container",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return cobraCmd
}

func (cmd *RemoveCmd) Run(ctx context.Context) error {
	// check if we already have built the image
	if isContainerRunning() {
		return fmt.Errorf("container is running, please stop first")
	}

	// add ignore paths
	buildIgnorePaths(nil)

	// delete the filesystem
	err := util.DeleteFilesystem()
	if err != nil {
		return fmt.Errorf("error deleting filesystem: %w", err)
	}

	return os.Remove(ImageConfigOutput)
}
