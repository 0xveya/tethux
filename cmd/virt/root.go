package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "virt",
		Short: "tethux virt - container/vm provider smoke tests",
	}

	cmd.AddCommand(
		smokeCmd(),
		listCmd(),
		pullCmd(),
		logsCmd(),
	)

	return cmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "tethux-virt: %v\n", err)
		os.Exit(1)
	}
}
