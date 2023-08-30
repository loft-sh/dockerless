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

	// fill parameters through env
	if cmd.Dockerfile == "" {
		cmd.Dockerfile = os.Getenv("DOCKERLESS_DOCKERFILE")
		if cmd.Dockerfile == "" {
			return fmt.Errorf("--dockerfile is missing")
		}
	}
	if cmd.Context == "" {
		cmd.Context = os.Getenv("DOCKERLESS_CONTEXT")
		if cmd.Context == "" {
			return fmt.Errorf("--context is missing")
		}
	}
	if cmd.Target == "" {
		cmd.Target = os.Getenv("DOCKERLESS_TARGET")
	}
	if len(cmd.BuildArgs) == 0 {
		buildArgs := os.Getenv("DOCKERLESS_BUILD_ARGS")
		if buildArgs != "" {
			_ = json.Unmarshal([]byte(buildArgs), &cmd.BuildArgs)
		}
	}

	// start actual build
	image, err := cmd.build()
	if err != nil {
		return err
	}

	// write config file to file
	configFile, err := image.ConfigFile()
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(ImageConfigOutput, out, 0666)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *BuildCmd) build() (v1.Image, error) {
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

	// change dir before building
	err = os.Chdir(cmd.Context)
	if err != nil {
		return nil, fmt.Errorf("change dir: %w", err)
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
		SnapshotMode:      "time",
		RunV2:             true,
		NoPush:            true,
		KanikoDir:         "/.dockerless",
		CacheRunLayers:    true,
		CacheCopyLayers:   true,
		CompressedCaching: true,
		SkipUnusedStages:  true,
		Compression:       config.ZStd,
		CompressionLevel:  3,
		CacheOptions: config.CacheOptions{
			CacheDir: "/.dockerless/cache",
			CacheTTL: time.Hour * 24 * 7,
		},
	})
	if err != nil {
		// add a passwd as other we won't be able to exec into this container
		_ = addPasswd()
		return nil, err
	}

	return image, err
}

func addPasswd() error {
	return os.WriteFile("/etc/passwd", []byte("root:x:0:0:root:/root:/.dockerless/bin/sh"), 0666)
}

func buildIgnorePaths(extraPaths []string) {
	// we need to add a couple of extra ignore paths for kaniko
	ignorePaths := append([]string{
		"/.dockerless",
		"/workspaces",
		"/etc/envfile.json",
		"/etc/resolv.conf",
		"/var/run",
	}, extraPaths...)
	for _, ignorePath := range ignorePaths {
		util.AddToDefaultIgnoreList(util.IgnoreListEntry{
			Path:            ignorePath,
			PrefixMatchOnly: false,
		})
	}
}
