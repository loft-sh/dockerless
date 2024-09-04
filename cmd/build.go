package cmd

import (
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
	Dockerfile  string
	Context     string
	Target      string
	Registry    string
	BuildArgs   []string
	IgnorePaths []string
	Insecure    bool
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
			return cmd.Run()
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Dockerfile, "dockerfile", "", "Dockerfile to build from.")
	cobraCmd.Flags().StringVar(&cmd.Target, "target", "", "The docker target stage to build.")
	cobraCmd.Flags().StringVar(&cmd.Context, "context", "", "Context to build from.")
	cobraCmd.Flags().StringArrayVar(&cmd.BuildArgs, "build-arg", []string{}, "Docker build args.")
	cobraCmd.Flags().StringArrayVar(&cmd.IgnorePaths, "ignore-path", []string{}, "Extra paths to exclude from deletion.")
	cobraCmd.Flags().BoolVar(&cmd.Insecure, "insecure", true, "If true will not check for certificates")
	cobraCmd.Flags().StringVar(&cmd.Registry, "registry-cache", "", "Registry to use as remote cache.")
	return cobraCmd
}

func (cmd *BuildCmd) Run() error {
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

	// parse extra build args
	buildArgs := os.Getenv("DOCKERLESS_BUILD_ARGS")
	if buildArgs != "" {
		extraBuildArgs := []string{}
		_ = json.Unmarshal([]byte(buildArgs), &extraBuildArgs)
		cmd.BuildArgs = append(cmd.BuildArgs, extraBuildArgs...)
	}

	// start actual build
	image, err := cmd.build()
	if err != nil {
		return err
	}

	// write config file to file
	configFile, err := image.ConfigFile()
	if err != nil {
		return fmt.Errorf("get image config: %w", err)
	}

	out, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal image config: %w", err)
	}

	err = os.WriteFile(ImageConfigOutput, out, 0666)
	if err != nil {
		return fmt.Errorf("write image config: %w", err)
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
	err = os.Chdir("/")
	if err != nil {
		return nil, fmt.Errorf("change dir: %w", err)
	}

	opts := &config.KanikoOptions{
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
		Cache:             true,
		CacheRunLayers:    true,
		CacheCopyLayers:   true,
		CompressedCaching: true,
		SkipUnusedStages:  true,
		Compression:       config.ZStd,
		CompressionLevel:  3,
		CacheOptions: config.CacheOptions{
			CacheTTL: time.Hour * 24 * 7,
		},
	}
	if cmd.Registry != "" {
		opts.CacheRepo = cmd.Registry
	} else {
		opts.CacheOptions.CacheDir = "/.dockerless/cache"
	}

	// let's build!
	image, err := executor.DoBuild(opts)
	if err != nil {
		// add a passwd as other we won't be able to exec into this container
		if addPwdErr := addPasswd(); addPwdErr != nil {
			return nil, fmt.Errorf("build and add passwd error occurred: %w --- %w", err, addPwdErr)
		}

		return nil, fmt.Errorf("build error: %w", err)
	}

	return image, nil
}

func addPasswd() error {
	err := os.WriteFile("/etc/passwd", []byte("root:x:0:0:root:/root:/.dockerless/bin/sh"), 0666)
	if err != nil {
		return fmt.Errorf("write passwd: %w", err)
	}

	return nil
}

func buildIgnorePaths(extraPaths []string) {
	// we need to add a couple of extra ignore paths for kaniko
	ignorePaths := append([]string{
		"/.dockerless",
		"/workspaces",
		"/etc/envfile.json",
		"/etc/resolv.conf",
		"/var/run",
		"/product_uuid",
	}, extraPaths...)
	for _, ignorePath := range ignorePaths {
		util.AddToDefaultIgnoreList(util.IgnoreListEntry{
			Path:            ignorePath,
			PrefixMatchOnly: false,
		})
	}
}
