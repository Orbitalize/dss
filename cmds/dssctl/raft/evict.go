package raft

import (
	"fmt"
	"strings"
	"time"

	"github.com/interuss/dss/pkg/admin/client"
	restapi "github.com/interuss/dss/pkg/api/adminv1"
	"github.com/interuss/stacktrace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// evictionGroupNames are the Raft groups with eviction, in the order in which they are processed.
var evictionGroupNames = []string{"rid", "scd"}

// entityFlag is an entity type of a Raft group, as selected by the flags of dssctl raft evict.
type entityFlag struct {
	// name is the prefix of its flags, as in db-manager evict.
	name  string
	group string
	// entityType is its name in the admin API.
	entityType string
}

var entityFlags = []entityFlag{
	{name: "rid_isa", group: "rid", entityType: "isas"},
	{name: "rid_sub", group: "rid", entityType: "subscriptions"},
	{name: "scd_oir", group: "scd", entityType: "operational_intents"},
	{name: "scd_sub", group: "scd", entityType: "subscriptions"},
}

var (
	evictCmd = &cobra.Command{
		Use:   "evict",
		Short: "Evict the expired entities of the Raft groups (rid, scd) of the DSS node",
		Long: `Evict the expired entities of the Raft groups (rid, scd) of the DSS node, through its admin API.

The leader of each group runs a cleanup job at the interval of its eviction settings, which are set with
"config set". Nothing is evicted until they are set. "run" runs a cleanup job now, independently of them.`,
	}

	evictRunCmd = &cobra.Command{
		Use:   "run",
		Short: "Run a cleanup job now on the Raft groups (rid, scd)",
		Long: `Run a cleanup job now on the Raft groups (rid, scd), with the same flags as db-manager evict except the batch
sizes, and independently of the eviction settings.

The expired entities are only counted unless --delete is set. A group with none of its entity types selected is
skipped.`,
		RunE: evictRun,
	}

	runFlagValues runFlags
)

func init() {
	runFlagValues.register(evictRunCmd.Flags())
	evictCmd.AddCommand(evictRunCmd, evictConfigCmd)
}

// runFlags are the flags of dssctl raft evict run.
type runFlags struct {
	selected   map[string]*bool
	ttls       map[string]*time.Duration
	limits     map[string]*int64
	deleteFlag bool
}

func (f *runFlags) register(fs *pflag.FlagSet) {
	f.selected = map[string]*bool{}
	for _, e := range entityFlags {
		f.selected[e.name] = fs.Bool(e.name, true, fmt.Sprintf("set this flag to true to evict the expired %s of the %s group", e.entityType, e.group))
	}
	f.ttls = map[string]*time.Duration{
		"rid": fs.Duration("rid_ttl", 30*time.Minute, "time-to-live duration used for determining RID entries expiration"),
		"scd": fs.Duration("scd_ttl", 2688*time.Hour, "time-to-live duration used for determining SCD entries expiration"),
	}
	f.limits = map[string]*int64{
		"rid": fs.Int64("rid_limit", 0, "maximum number of RID entities deleted per entity type, 0 for unlimited (requires --delete)"),
		"scd": fs.Int64("scd_limit", 0, "maximum number of SCD entities deleted per entity type, 0 for unlimited (requires --delete)"),
	}
	fs.BoolVar(&f.deleteFlag, "delete", false, "set this flag to true to delete the expired entities, otherwise they are only counted")
}

// parameters returns the parameters of the cleanup job of each group with at least one entity type selected.
func (f *runFlags) parameters(fs *pflag.FlagSet) (map[string]*restapi.EvictionRunParameters, error) {
	if !f.deleteFlag {
		for _, group := range evictionGroupNames {
			if fs.Changed(group + "_limit") {
				return nil, stacktrace.NewError("--%s_limit requires --delete", group)
			}
		}
	}

	result := map[string]*restapi.EvictionRunParameters{}
	for _, e := range entityFlags {
		if !*f.selected[e.name] {
			continue
		}
		params, ok := result[e.group]
		if !ok {
			params = &restapi.EvictionRunParameters{Entities: []restapi.EntityEviction{}, Delete: new(f.deleteFlag)}
			result[e.group] = params
		}
		entity := restapi.EntityEviction{Type: e.entityType, Ttl: f.ttls[e.group].String()}
		if f.deleteFlag {
			entity.Limit = new(*f.limits[e.group])
		}
		params.Entities = append(params.Entities, entity)
	}
	return result, nil
}

func evictRun(cmd *cobra.Command, _ []string) error {
	params, err := runFlagValues.parameters(cmd.Flags())
	if err != nil {
		return err
	}
	var names []string
	for _, name := range evictionGroupNames {
		if _, ok := params[name]; ok {
			names = append(names, name)
		}
	}
	groups, err := newGroups(names)
	if err != nil {
		return err
	}

	return forEachGroup(groups, func(g namedGroup) (string, error) {
		result, err := g.RunEviction(cmd.Context(), params[g.name])
		if err != nil {
			return "", err
		}
		return formatRunResult(result, runFlagValues.deleteFlag), nil
	})
}

func formatRunResult(result *restapi.EvictionRunResult, deleted bool) string {
	counts := make([]string, 0, len(result.Counts))
	for _, c := range result.Counts {
		counts = append(counts, fmt.Sprintf("%d %s", c.Count, c.Type))
	}
	if deleted {
		return "deleted " + strings.Join(counts, ", ")
	}
	return "found expired " + strings.Join(counts, ", ") + " (not deleted, use --delete to delete them)"
}

// newGroups returns the named Raft groups of the DSS node, in the order of names.
func newGroups(names []string) ([]namedGroup, error) {
	if len(names) == 0 {
		return nil, stacktrace.NewError("no Raft group selected")
	}
	result := make([]namedGroup, 0, len(names))
	for _, name := range names {
		group, err := client.NewRaftGroup(adminSocket, name, adminTimeout)
		if err != nil {
			return nil, stacktrace.Propagate(err, "invalid --admin_socket")
		}
		result = append(result, namedGroup{name: name, RaftGroup: group})
	}
	return result, nil
}
