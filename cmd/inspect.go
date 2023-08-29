package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
)

type InspectCmd struct{}

// NewInspectCmd returns a new command
func NewInspectCmd() *cobra.Command {
	cmd := &InspectCmd{}
	cobraCmd := &cobra.Command{
		Use:           "inspect",
		Short:         "Inspects the container",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	return cobraCmd
}

func (cmd *InspectCmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please specify either 'container' or 'image'")
	}

	if args[0] == "container" {
		// load container.json
		out, err := os.ReadFile(ContainerConfigOutput)
		if err != nil {
			return printObj(&[]ContainerDetails{})
		}

		// parse config file
		configFile := &v1.ConfigFile{}
		err = json.Unmarshal(out, configFile)
		if err != nil {
			return fmt.Errorf("parse container config: %w", err)
		}

		state := "exited"
		if isContainerRunning() {
			state = "running"
		}
		return printObj(&[]ContainerDetails{
			{
				ID:     configFile.Container,
				Config: configFile.Config,
				State: ContainerDetailsState{
					Status:    state,
					StartedAt: configFile.Created.String(),
				},
			},
		})
	} else if args[0] == "image" {
		// load image.json
		out, err := os.ReadFile(ImageConfigOutput)
		if err != nil {
			return printObj(&[]ImageDetails{})
		}

		// parse config file
		configFile := &v1.ConfigFile{}
		err = json.Unmarshal(out, configFile)
		if err != nil {
			return fmt.Errorf("parse image config: %w", err)
		}

		return printObj(&[]ImageDetails{
			{
				ID:     configFile.Container,
				Config: configFile.Config,
			},
		})
	} else {
		return fmt.Errorf("please specify either 'container' or 'image'")
	}

	return nil
}

func printObj(obj interface{}) error {
	out, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("error marshal: %w", err)
	}

	fmt.Print(string(out))
	return nil
}

type ImageDetails struct {
	ID     string
	Config v1.Config
}

type ContainerDetails struct {
	ID      string                `json:"ID,omitempty"`
	Created string                `json:"Created,omitempty"`
	State   ContainerDetailsState `json:"State,omitempty"`
	Config  v1.Config             `json:"Config,omitempty"`
}

type ContainerDetailsState struct {
	Status    string `json:"Status,omitempty"`
	StartedAt string `json:"StartedAt,omitempty"`
}
