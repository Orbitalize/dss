package raft

import (
	"fmt"
	"slices"
	"strings"
	"time"

	restapi "github.com/interuss/dss/pkg/api/adminv1"
	"github.com/interuss/stacktrace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	evictConfigCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage the eviction settings of the Raft groups (rid, scd)",
	}

	evictConfigGetCmd = &cobra.Command{
		Use:   "get",
		Short: "Print the eviction settings of the Raft groups (rid, scd)",
		Long:  "Print the eviction settings of the Raft groups selected with --groups, rid and scd unless set: aux has no eviction.",
		RunE:  evictConfigGet,
	}

	evictConfigSetCmd = &cobra.Command{
		Use:   "set",
		Short: "Change the eviction settings of the Raft groups (rid, scd)",
		Long: `Change the eviction settings of the Raft groups (rid, scd): only the settings given are changed, in the groups
they belong to.

Giving the TTL of an entity type enables its eviction, or replaces its TTL and limit (0, i.e. unlimited, unless
given). --disable disables the eviction of entity types. The interval of a group is required to set its eviction
settings for the first time.

The current settings are read, changed and written back as a whole: changes made at the same time by someone else
may be lost.`,
		RunE: evictConfigSet,
	}

	configSetFlagValues configSetFlags
)

func init() {
	configSetFlagValues.register(evictConfigSetCmd.Flags())
	evictConfigCmd.AddCommand(evictConfigGetCmd, evictConfigSetCmd)
}

// configSetFlags are the flags of dssctl raft evict config set.
type configSetFlags struct {
	intervals map[string]*time.Duration
	ttls      map[string]*time.Duration
	limits    map[string]*int64
	disable   []string
}

func (f *configSetFlags) register(fs *pflag.FlagSet) {
	f.intervals = map[string]*time.Duration{}
	for _, group := range evictionGroupNames {
		f.intervals[group] = fs.Duration(group+"_interval", 0, fmt.Sprintf("time between two cleanup jobs of the %s group", group))
	}
	f.ttls = map[string]*time.Duration{}
	f.limits = map[string]*int64{}
	var names []string
	for _, e := range entityFlags {
		f.ttls[e.name] = fs.Duration(e.name+"_ttl", 0, fmt.Sprintf("time-to-live duration of the %s of the %s group, enabling their eviction", e.entityType, e.group))
		f.limits[e.name] = fs.Int64(e.name+"_limit", 0, fmt.Sprintf("maximum number of %s of the %s group deleted per cleanup job, 0 for unlimited (requires --%s_ttl)", e.entityType, e.group, e.name))
		names = append(names, e.name)
	}
	fs.StringSliceVar(&f.disable, "disable", nil, "comma-separated entity types whose eviction is disabled, among "+strings.Join(names, ","))
}

// configChange is the change of the eviction settings of a group requested by the flags.
type configChange struct {
	interval *time.Duration
	// set maps the entity types enabled or replaced to their parameters.
	set map[string]restapi.EntityEviction
	// disable are the entity types disabled.
	disable []string
}

// changes returns the change of each group with at least one setting changed by the flags.
func (f *configSetFlags) changes(fs *pflag.FlagSet) (map[string]*configChange, error) {
	for _, name := range f.disable {
		if !slices.ContainsFunc(entityFlags, func(e entityFlag) bool { return e.name == name }) {
			return nil, stacktrace.NewError("unknown entity type %q in --disable", name)
		}
	}

	result := map[string]*configChange{}
	change := func(group string) *configChange {
		if c, ok := result[group]; ok {
			return c
		}
		c := &configChange{set: map[string]restapi.EntityEviction{}}
		result[group] = c
		return c
	}
	for _, group := range evictionGroupNames {
		if fs.Changed(group + "_interval") {
			change(group).interval = f.intervals[group]
		}
	}
	for _, e := range entityFlags {
		ttlChanged := fs.Changed(e.name + "_ttl")
		disabled := slices.Contains(f.disable, e.name)
		switch {
		case fs.Changed(e.name+"_limit") && !ttlChanged:
			return nil, stacktrace.NewError("--%s_limit requires --%s_ttl", e.name, e.name)
		case ttlChanged && disabled:
			return nil, stacktrace.NewError("--%s_ttl and --disable %s are exclusive", e.name, e.name)
		case ttlChanged:
			change(e.group).set[e.entityType] = restapi.EntityEviction{Type: e.entityType, Ttl: f.ttls[e.name].String(), Limit: new(*f.limits[e.name])}
		case disabled:
			change(e.group).disable = append(change(e.group).disable, e.entityType)
		}
	}
	if len(result) == 0 {
		return nil, stacktrace.NewError("no eviction setting given")
	}
	return result, nil
}

// apply returns the eviction settings of group resulting from c applied to current, nil if they were never set.
func (c *configChange) apply(group string, current *restapi.EvictionConfig) (*restapi.EvictionConfig, error) {
	result := &restapi.EvictionConfig{Entities: []restapi.EntityEviction{}}
	if current != nil {
		result.Interval = current.Interval
		for _, e := range current.Entities {
			if _, replaced := c.set[e.Type]; !replaced && !slices.Contains(c.disable, e.Type) {
				result.Entities = append(result.Entities, e)
			}
		}
	}
	if c.interval != nil {
		result.Interval = c.interval.String()
	}
	if result.Interval == "" {
		return nil, stacktrace.NewError("--%s_interval is required to set the eviction settings of the %s group for the first time", group, group)
	}
	for _, e := range entityFlags {
		if params, ok := c.set[e.entityType]; ok && e.group == group {
			result.Entities = append(result.Entities, params)
		}
	}
	return result, nil
}

func evictConfigSet(cmd *cobra.Command, _ []string) error {
	changes, err := configSetFlagValues.changes(cmd.Flags())
	if err != nil {
		return err
	}
	var names []string
	for _, name := range evictionGroupNames {
		if _, ok := changes[name]; ok {
			names = append(names, name)
		}
	}
	groups, err := newGroups(names)
	if err != nil {
		return err
	}

	return forEachGroup(groups, func(g namedGroup) (string, error) {
		current, err := g.EvictionConfig(cmd.Context())
		if err != nil {
			return "", err
		}
		config, err := changes[g.name].apply(g.name, current)
		if err != nil {
			return "", err
		}
		stored, err := g.SetEvictionConfig(cmd.Context(), config)
		if err != nil {
			return "", err
		}
		return formatEvictionConfig(g.name, stored), nil
	})
}

func evictConfigGet(cmd *cobra.Command, _ []string) error {
	selected, err := selectedGroups()
	if err != nil {
		return err
	}
	var groups []namedGroup
	for _, g := range selected {
		if slices.Contains(evictionGroupNames, g.name) {
			groups = append(groups, g)
		}
	}
	if len(groups) == 0 {
		return stacktrace.NewError("no Raft group with eviction selected with --groups, expected some of %s", strings.Join(evictionGroupNames, ","))
	}

	return forEachGroup(groups, func(g namedGroup) (string, error) {
		config, err := g.EvictionConfig(cmd.Context())
		if err != nil {
			return "", err
		}
		return formatEvictionConfig(g.name, config), nil
	})
}

// formatEvictionConfig formats config, every entity type of group included.
func formatEvictionConfig(group string, config *restapi.EvictionConfig) string {
	if config == nil {
		return "not configured, nothing is evicted"
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "\n  interval: %s", config.Interval)
	for _, e := range entityFlags {
		if e.group != group {
			continue
		}
		i := slices.IndexFunc(config.Entities, func(c restapi.EntityEviction) bool { return c.Type == e.entityType })
		if i < 0 {
			fmt.Fprintf(&builder, "\n  %s: disabled", e.entityType)
			continue
		}
		limit := "unlimited"
		if l := config.Entities[i].Limit; l != nil && *l != 0 {
			limit = fmt.Sprint(*l)
		}
		fmt.Fprintf(&builder, "\n  %s: ttl %s, limit %s", e.entityType, config.Entities[i].Ttl, limit)
	}
	return builder.String()
}
