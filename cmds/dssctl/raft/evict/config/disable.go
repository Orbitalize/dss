package config

import (
	"github.com/interuss/dss/cmds/dssctl/raft/raftgroup"
	"github.com/spf13/cobra"
)

var (
	disableCmd = &cobra.Command{
		Use:   "disable",
		Short: "Disable the scheduled cleanup jobs of the Raft groups",
		Long:  "Disable the scheduled cleanup jobs of the Raft groups selected with --groups or both rid and scd if unset.",
		RunE:  disable,
	}

	enableCmd = &cobra.Command{
		Use:   "enable",
		Short: "Enable the scheduled cleanup jobs of the Raft groups",
		Long:  "Enable the scheduled cleanup jobs of the Raft groups selected with --groups or both rid and scd if unset.",
		RunE:  enable,
	}
)

func disable(cmd *cobra.Command, _ []string) error {
	groups, err := raftgroup.SelectedEviction(cmd.Flags())
	if err != nil {
		return err
	}

	return raftgroup.ForEach(groups, func(g raftgroup.Group) (string, error) {
		config, err := g.DisableEviction(cmd.Context())
		if err != nil {
			return "", err
		}
		return formatConfig(config), nil
	})
}

func enable(cmd *cobra.Command, _ []string) error {
	groups, err := raftgroup.SelectedEviction(cmd.Flags())
	if err != nil {
		return err
	}

	return raftgroup.ForEach(groups, func(g raftgroup.Group) (string, error) {
		config, err := g.EnableEviction(cmd.Context())
		if err != nil {
			return "", err
		}
		return formatConfig(config), nil
	})
}
