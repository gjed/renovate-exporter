package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "renovate-exporter",
		Short: "Prometheus exporter for Renovate dependency update metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&configFile, "config", "config.yaml", "path to config file")

	return cmd
}
