// Package raft implements the dssctl commands managing the membership of the Raft groups of a DSS instance.
package raft

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"github.com/interuss/dss/pkg/admin"
	"github.com/interuss/dss/pkg/admin/client"
	"github.com/interuss/stacktrace"
	"github.com/spf13/cobra"
)

// groupNames are the Raft groups of a DSS instance, in the order in which they are processed.
var groupNames = []string{"aux", "rid", "scd"}

var (
	RaftCmd = &cobra.Command{
		Use:   "raft",
		Short: "Manage the membership of the Raft groups (aux, rid, scd) of a DSS instance",
		Long: `Manage the membership of the Raft groups (aux, rid, scd) of a DSS instance, through its admin API (see the --admin_addr flag of the core-service).

A change is requested to each selected group independently: it may succeed for some groups and fail for others,
in which case it can be retried for the failed groups only with --groups.`,
		// Once the flags are valid, errors come from the requests: the usage would not help.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmd.ValidateRequiredFlags(); err != nil {
				return err
			}
			cmd.SilenceUsage = true
			return nil
		},
	}

	groups []string
)

func init() {
	RaftCmd.PersistentFlags().StringSliceVar(&groups, "groups", groupNames, "Comma-separated Raft groups to act on, all of them if not set")
	RaftCmd.AddCommand(addCmd, removeCmd, listCmd)
}

// namedGroup is a Raft group of the DSS instance, managed through its admin API.
type namedGroup struct {
	name string
	admin.RaftGroup
}

// selectedGroups returns the Raft groups selected with --groups, in processing order.
func selectedGroups() ([]namedGroup, error) {
	for _, name := range groups {
		if !slices.Contains(groupNames, name) {
			return nil, stacktrace.NewError("unknown Raft group %q in --groups, expected some of %s", name, strings.Join(groupNames, ","))
		}
	}

	var result []namedGroup
	for _, name := range groupNames {
		if !slices.Contains(groups, name) {
			continue
		}
		group, err := client.InitRaftGroup(name)
		if err != nil {
			return nil, stacktrace.Propagate(err, "failed to initialize the %s Raft group", name)
		}
		result = append(result, namedGroup{name: name, RaftGroup: group})
	}
	if len(result) == 0 {
		return nil, stacktrace.NewError("no Raft group selected with --groups")
	}
	return result, nil
}

// ErrGroupsFailed is returned when a command failed for some Raft groups, whose errors were already reported.
var ErrGroupsFailed = errors.New("failed for some Raft groups")

// forEachGroup calls fn for every group concurrently, then reports their outcomes in out, in the order of groups.
// It processes all the groups even if some fail, in which case it returns ErrGroupsFailed.
func forEachGroup(out io.Writer, groups []namedGroup, fn func(g namedGroup) (string, error)) error {
	type result struct {
		msg string
		err error
	}
	// every goroutine only writes its own result, so no lock is needed
	results := make([]result, len(groups))

	var wg sync.WaitGroup
	for i, g := range groups {
		wg.Go(func() {
			msg, err := fn(g)
			results[i] = result{msg: msg, err: err}
		})
	}
	wg.Wait()

	failed := false
	for i, g := range groups {
		if err := results[i].err; err != nil {
			failed = true
			_, _ = fmt.Fprintf(out, "%s: error: %#s\n", g.name, err)
			continue
		}
		_, _ = fmt.Fprintf(out, "%s: %s\n", g.name, results[i].msg)
	}

	if failed {
		return ErrGroupsFailed
	}
	return nil
}
