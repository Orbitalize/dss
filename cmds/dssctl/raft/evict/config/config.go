package config

import (
	"fmt"
	"strconv"
	"strings"

	restapi "github.com/interuss/dss/pkg/api/adminv1"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage the eviction settings of the Raft groups",
}

func init() {
	Cmd.AddCommand(getCmd, setCmd, disableCmd, enableCmd)
}

func formatConfig(config *restapi.EvictionConfig) string {
	interval := config.Interval
	if config.Disabled != nil && *config.Disabled {
		interval += " (scheduled cleanup jobs disabled)"
	}
	limit := "unlimited"
	if l := config.Parameters.Limit; l != nil && *l != 0 {
		limit = strconv.Itoa(int(*l))
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "\n  interval: %s", interval)
	fmt.Fprintf(&builder, "\n  ttl: %s", config.Parameters.Ttl)
	fmt.Fprintf(&builder, "\n  limit: %s", limit)
	return builder.String()
}
