package cmd

import "github.com/spf13/cobra"

// NewRootCmd returns a new root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "dockerless",
		Short:         "Dockerless",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(NewBuildCmd())
	rootCmd.AddCommand(NewStartCmd())
	rootCmd.AddCommand(NewExecCmd())
	rootCmd.AddCommand(NewStopCmd())
	rootCmd.AddCommand(NewRemoveCmd())
	rootCmd.AddCommand(NewInspectCmd())
	return rootCmd
}
