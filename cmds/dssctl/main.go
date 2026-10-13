package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/interuss/dss/cmds/dssctl/raft"
	"github.com/spf13/cobra"
)

var (
	dssctlCmd = &cobra.Command{
		Use:   "dssctl",
		Short: "DSS instance admin command line tool",
		// The error is printed by main.
		SilenceErrors: true,
	}
)

func init() {
	dssctlCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	dssctlCmd.AddCommand(raft.RaftCmd)
}

func main() {
	if err := dssctlCmd.Execute(); err != nil {
		// The errors of the Raft groups were already reported along with the results of the other groups.
		if !errors.Is(err, raft.ErrGroupsFailed) {
			// %#s formats stacktrace errors without the stack.
			fmt.Fprintf(os.Stderr, "Error: %#s\n", err)
		}
		os.Exit(1)
	}
}
