package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/containerd/containerd/platforms"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
)

var ImageConfigOutput = "/.dockerless/image.json"

type BuildCmd struct {
	Dockerfile string
	Context    string
	BuildArgs  []string

	Target      string
	Insecure    bool
	IgnorePaths []string
}

// NewBuildCmd returns a new build command
func NewBuildCmd() *cobra.Command {
	cmd := &BuildCmd{}
	cobraCmd := &cobra.Command{
		Use:           "build",
		Short:         "Build the container",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Dockerfile, "dockerfile", "", "Dockerfile to build from.")
	cobraCmd.Flags().StringVar(&cmd.Target, "target", "", "The docker target stage to build.")
	cobraCmd.Flags().StringVar(&cmd.Context, "context", "", "Context to build from.")
	cobraCmd.Flags().StringArrayVar(&cmd.BuildArgs, "build-arg", []string{}, "Docker build args.")
	cobraCmd.Flags().StringArrayVar(&cmd.IgnorePaths, "ignore-path", []string{}, "Extra paths to exclude from deletion.")
	cobraCmd.Flags().BoolVar(&cmd.Insecure, "insecure", true, "If true will not check for certificates")
	return cobraCmd
}

func (cmd *BuildCmd) Run(ctx context.Context) error {
	// check if we already have built the image
	_, err := os.Stat(ImageConfigOutput)
	if err == nil {
		fmt.Println("skip building, because image is already built")
		return nil
	}

	image, err := cmd.build(ctx)
	if err != nil {
		return err
	}

	configFile, err := image.ConfigFile()
	if err != nil {
		return err
	}

	out, err := json.Marshal(configFile)
	if err != nil {
		return err
	}

	err = os.WriteFile(ImageConfigOutput, out, 0666)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *BuildCmd) build(ctx context.Context) (v1.Image, error) {
	// add ignore paths
	buildIgnorePaths(cmd.IgnorePaths)

	// make sure we detect the correct ignore list
	err := util.InitIgnoreList(true)
	if err != nil {
		return nil, fmt.Errorf("init ignore list: %w", err)
	}

	// make sure to delete previous contents
	err = util.DeleteFilesystem()
	if err != nil {
		return nil, fmt.Errorf("delete filesystem: %w", err)
	}

	// let's build!
	image, err := executor.DoBuild(&config.KanikoOptions{
		Destinations:   []string{"local"},
		Unpack:         true,
		BuildArgs:      cmd.BuildArgs,
		DockerfilePath: cmd.Dockerfile,
		RegistryOptions: config.RegistryOptions{
			Insecure:      cmd.Insecure,
			InsecurePull:  cmd.Insecure,
			SkipTLSVerify: cmd.Insecure,
		},
		SrcContext:        cmd.Context,
		Target:            cmd.Target,
		CustomPlatform:    platforms.Format(platforms.Normalize(platforms.DefaultSpec())),
		SnapshotMode:      "redo",
		RunV2:             true,
		NoPush:            true,
		KanikoDir:         "/.dockerless",
		CacheRunLayers:    true,
		CacheCopyLayers:   true,
		CompressedCaching: true,
		Compression:       config.ZStd,
		CompressionLevel:  3,
		CacheOptions: config.CacheOptions{
			CacheTTL: time.Hour * 24 * 7,
		},
	})
	if err != nil {
		return nil, err
	}

	return image, err
}

func buildIgnorePaths(extraPaths []string) {
	// we need to add a couple of extra ignore paths for kaniko
	ignorePaths := append([]string{
		"/.dockerless",
		"/workspaces",
		"/etc/resolv.conf",
	}, extraPaths...)
	for _, ignorePath := range ignorePaths {
		util.AddToDefaultIgnoreList(util.IgnoreListEntry{
			Path:            ignorePath,
			PrefixMatchOnly: false,
		})
	}
}
