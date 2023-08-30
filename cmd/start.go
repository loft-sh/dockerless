package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
)

var ContainerConfigOutput = "/.dockerless/container.json"

var ContainerPID = "/.dockerless/pid"

type StartCmd struct {
	Entrypoint []string
	Cmd        []string

	User   string
	Env    []string
	Labels []string

	Wait bool
}

// NewStartCmd returns a new start command
func NewStartCmd() *cobra.Command {
	cmd := &StartCmd{}
	cobraCmd := &cobra.Command{
		Use:           "start",
		Short:         "Starts the container",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	cobraCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "If true, will wait until the container is built.")
	cobraCmd.Flags().StringVar(&cmd.User, "user", "", "The container user to run the entrypoint with.")
	cobraCmd.Flags().StringArrayVar(&cmd.Entrypoint, "entrypoint", []string{}, "The entrypoint to use.")
	cobraCmd.Flags().StringArrayVar(&cmd.Cmd, "cmd", []string{}, "The cmds to use.")
	cobraCmd.Flags().StringArrayVar(&cmd.Env, "env", []string{}, "Extra environment variables to start the container with.")
	cobraCmd.Flags().StringArrayVar(&cmd.Labels, "labels", []string{}, "Labels to add to the container.")
	return cobraCmd
}

func (cmd *StartCmd) Run(ctx context.Context) error {
	// check if we already have built the image
	if isContainerRunning() {
		return fmt.Errorf("container is already running")
	}

	// get the built image config
	var (
		out []byte
		err error
	)
	for {
		out, err = os.ReadFile(ImageConfigOutput)
		if err != nil {
			if !cmd.Wait {
				return err
			}

			time.Sleep(time.Second)
			continue
		}

		break
	}

	// unmarshal the config file
	configFile := &v1.ConfigFile{}
	err = json.Unmarshal(out, configFile)
	if err != nil {
		return err
	}

	// entrypoint
	if len(cmd.Entrypoint) > 0 {
		configFile.Config.Entrypoint = cmd.Entrypoint
	}

	// cmd
	if len(cmd.Cmd) > 0 {
		configFile.Config.Cmd = cmd.Cmd
	}

	// add labels
	if len(cmd.Labels) > 0 {
		if configFile.Config.Labels == nil {
			configFile.Config.Labels = map[string]string{}
		}

		for _, label := range cmd.Labels {
			splitted := strings.SplitN(label, "=", 2)
			if len(splitted) == 2 {
				configFile.Config.Labels[splitted[0]] = splitted[1]
			}
		}
	}

	// add environment variables
	configFile.Config.Env = append(configFile.Config.Env, cmd.Env...)

	// user
	if cmd.User != "" {
		configFile.Config.User = cmd.User
	}

	// set to root if not specified
	if configFile.Config.WorkingDir == "" {
		configFile.Config.WorkingDir = "/"
	} else {
		// create working dir
		err = os.MkdirAll(configFile.Config.WorkingDir, 0777)
		if err != nil {
			return fmt.Errorf("create workspace folder: %w", err)
		}
	}

	// get user info
	homeDir, credential, err := getUserInfo(configFile.Config.User)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// prepare the command to run
	entrypoint := []string{}
	entrypoint = append(entrypoint, configFile.Config.Entrypoint...)
	entrypoint = append(entrypoint, configFile.Config.Cmd...)
	if len(entrypoint) == 0 {
		return fmt.Errorf("no entrypoint specified")
	}

	// build the container env
	containerEnv, err := getContainerEnv(homeDir, configFile.Config.Env)
	if err != nil {
		return fmt.Errorf("get container env: %w", err)
	}

	// write container config
	out, err = json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config file: %w", err)
	}

	err = os.WriteFile(ContainerConfigOutput, out, 0666)
	if err != nil {
		return fmt.Errorf("write container config: %w", err)
	}

	err = os.WriteFile(ContainerPID, []byte(strconv.Itoa(os.Getpid())), 0666)
	if err != nil {
		return fmt.Errorf("write container pid: %w", err)
	}

	// set container env before execution so we find executables
	for _, env := range containerEnv {
		splitted := strings.SplitN(env, "=", 2)
		if len(splitted) != 2 {
			continue
		}

		_ = os.Setenv(splitted[0], splitted[1])
	}

	// start command
	containerCmd := exec.Command(entrypoint[0], entrypoint[1:]...)
	containerCmd.SysProcAttr = &syscall.SysProcAttr{}
	containerCmd.SysProcAttr.Credential = credential
	containerCmd.Stdout = os.Stdout
	containerCmd.Stderr = os.Stderr
	containerCmd.Stdin = os.Stdin
	containerCmd.Dir = configFile.Config.WorkingDir
	containerCmd.Env = containerEnv
	return containerCmd.Run()
}

func isContainerRunning() bool {
	pid, err := os.ReadFile(ContainerPID)
	if err == nil {
		isRunning, _ := isProcessRunning(string(pid))
		if isRunning {
			return true
		}
	}

	return false
}

func isProcessRunning(pid string) (bool, error) {
	parsedPid, err := strconv.Atoi(pid)
	if err != nil {
		return false, err
	}

	process, err := os.FindProcess(parsedPid)
	if err != nil {
		return false, err
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false, nil
	}

	return true, nil
}

func setPath(containerEnvs []string) {
	for _, containerEnv := range containerEnvs {
		splitted := strings.SplitN(containerEnv, "=", 2)
		if len(splitted) == 2 && strings.ToUpper(splitted[0]) == "PATH" {
			_ = os.Setenv(splitted[0], splitted[1])
		}
	}
}

func getContainerEnv(homeDir string, containerEnv []string) ([]string, error) {
	env := []string{}
	env = append(env, "HOME="+homeDir)
	env = append(env, containerEnv...)
	setPath(env)
	return env, nil
}

func getUserInfo(name string) (string, *syscall.Credential, error) {
	userObj, err := getUser(name)
	if err != nil {
		return "", nil, err
	}

	uid, err := strconv.Atoi(userObj.Uid)
	if err != nil {
		return "", nil, fmt.Errorf("parse uid: %w", err)
	}

	gid, err := strconv.Atoi(userObj.Gid)
	if err != nil {
		return "", nil, fmt.Errorf("parse gid: %w", err)
	}

	intGroupIDs := []uint32{}
	groupIDs, err := userObj.GroupIds()
	if err == nil {
		for _, groupID := range groupIDs {
			parsed, err := strconv.Atoi(groupID)
			if err == nil {
				intGroupIDs = append(intGroupIDs, uint32(parsed))
			}
		}
	}

	homeDir := userObj.HomeDir
	if homeDir == "" {
		homeDir = "/root"
	}

	return homeDir, &syscall.Credential{
		Uid:    uint32(uid),
		Gid:    uint32(gid),
		Groups: intGroupIDs,
	}, nil
}

func getUser(name string) (*user.User, error) {
	if name == "" {
		name = "root"
	}

	// get user
	_, err := strconv.Atoi(name)
	if err == nil {
		return user.LookupId(name)
	}
	return user.Lookup(name)
}
