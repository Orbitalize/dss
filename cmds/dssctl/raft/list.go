package raft

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/interuss/dss/pkg/admin"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the members of the Raft groups of the DSS instance",
	Long:  "List the members of the Raft groups of the DSS instance, all of them unless selected with --groups, as known by the node serving each admin API.",
	RunE:  list,
}

func list(cmd *cobra.Command, _ []string) error {
	groups, err := selectedGroups()
	if err != nil {
		return err
	}

	return forEachGroup(cmd.OutOrStdout(), groups, func(g namedGroup) (string, error) {
		members, err := g.Members(cmd.Context())
		if err != nil {
			return "", err
		}

		return formatMembers(members), nil
	})
}

// formatMembers formats m as a table of the members, followed by the removed IDs.
func formatMembers(m admin.Membership) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "as known by node %d\n", m.NodeID)
	if m.LeaderID == 0 {
		builder.WriteString("  no leader known\n")
	}

	table := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "  ID\tURL\tROLE")
	for _, id := range slices.Sorted(maps.Keys(m.Members)) {
		fmt.Fprintf(table, "  %d\t%s\t%s\n", id, m.Members[id], memberRole(m, id))
	}
	_ = table.Flush()

	if len(m.RemovedIDs) > 0 {
		fmt.Fprintf(&builder, "  removed IDs: %v\n", slices.Sorted(slices.Values(m.RemovedIDs)))
	}
	return strings.TrimSuffix(builder.String(), "\n")
}

func memberRole(m admin.Membership, id uint64) string {
	switch {
	case id == m.LeaderID:
		return "leader"
	case slices.Contains(m.Voters, id):
		return "voter"
	default:
		return ""
	}
}
