package raft

import (
	"testing"

	restapi "github.com/interuss/dss/pkg/api/adminv1"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func parseRunFlags(t *testing.T, args ...string) (map[string]*restapi.EvictionRunParameters, error) {
	t.Helper()
	fs := pflag.NewFlagSet("run", pflag.ContinueOnError)
	var f runFlags
	f.register(fs)
	require.NoError(t, fs.Parse(args))
	return f.parameters(fs)
}

func TestRunParameters(t *testing.T) {
	t.Run("defaults count every entity type with the db-manager TTLs", func(t *testing.T) {
		params, err := parseRunFlags(t)
		require.NoError(t, err)
		require.Equal(t, map[string]*restapi.EvictionRunParameters{
			"rid": {Entities: []restapi.EntityEviction{{Type: "isas", Ttl: "30m0s"}, {Type: "subscriptions", Ttl: "30m0s"}}, Delete: new(false)},
			"scd": {Entities: []restapi.EntityEviction{{Type: "operational_intents", Ttl: "2688h0m0s"}, {Type: "subscriptions", Ttl: "2688h0m0s"}}, Delete: new(false)},
		}, params)
	})

	t.Run("delete with a limit, a group without entity type skipped", func(t *testing.T) {
		params, err := parseRunFlags(t, "--scd_oir=false", "--scd_sub=false", "--rid_sub=false", "--rid_ttl=1s", "--rid_limit=2", "--delete")
		require.NoError(t, err)
		require.Equal(t, map[string]*restapi.EvictionRunParameters{
			"rid": {Entities: []restapi.EntityEviction{{Type: "isas", Ttl: "1s", Limit: new(int64(2))}}, Delete: new(true)},
		}, params)
	})

	t.Run("limit without delete", func(t *testing.T) {
		_, err := parseRunFlags(t, "--scd_limit=2")
		require.Error(t, err)
	})
}

func parseConfigSetFlags(t *testing.T, args ...string) (map[string]*configChange, error) {
	t.Helper()
	fs := pflag.NewFlagSet("set", pflag.ContinueOnError)
	var f configSetFlags
	f.register(fs)
	require.NoError(t, fs.Parse(args))
	return f.changes(fs)
}

func TestConfigSetChanges(t *testing.T) {
	changes, err := parseConfigSetFlags(t, "--rid_interval=1h", "--rid_isa_ttl=30m", "--disable=scd_sub")
	require.NoError(t, err)
	require.Len(t, changes, 2, "only the groups with a changed setting")
	require.Equal(t, map[string]restapi.EntityEviction{"isas": {Type: "isas", Ttl: "30m0s", Limit: new(int64(0))}}, changes["rid"].set)
	require.Equal(t, []string{"subscriptions"}, changes["scd"].disable)

	for name, args := range map[string][]string{
		"nothing given":       {},
		"limit without TTL":   {"--rid_isa_limit=2"},
		"TTL and disable":     {"--rid_isa_ttl=1h", "--disable=rid_isa"},
		"unknown entity type": {"--disable=rid_constraints"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseConfigSetFlags(t, args...)
			require.Error(t, err)
		})
	}
}

func TestConfigChangeApply(t *testing.T) {
	current := &restapi.EvictionConfig{
		Interval: "1h0m0s",
		Entities: []restapi.EntityEviction{
			{Type: "isas", Ttl: "30m0s", Limit: new(int64(5))},
			{Type: "subscriptions", Ttl: "30m0s", Limit: new(int64(0))},
		},
	}

	t.Run("first time without interval", func(t *testing.T) {
		changes, err := parseConfigSetFlags(t, "--rid_isa_ttl=30m")
		require.NoError(t, err)
		_, err = changes["rid"].apply("rid", nil)
		require.Error(t, err)
	})

	t.Run("first time", func(t *testing.T) {
		changes, err := parseConfigSetFlags(t, "--rid_interval=1h", "--rid_sub_ttl=2h")
		require.NoError(t, err)
		got, err := changes["rid"].apply("rid", nil)
		require.NoError(t, err)
		require.Equal(t, &restapi.EvictionConfig{
			Interval: "1h0m0s",
			Entities: []restapi.EntityEviction{{Type: "subscriptions", Ttl: "2h0m0s", Limit: new(int64(0))}},
		}, got, "only the entity types given are enabled")
	})

	t.Run("interval only keeps the entity types", func(t *testing.T) {
		changes, err := parseConfigSetFlags(t, "--rid_interval=2h")
		require.NoError(t, err)
		got, err := changes["rid"].apply("rid", current)
		require.NoError(t, err)
		require.Equal(t, "2h0m0s", got.Interval)
		require.Equal(t, current.Entities, got.Entities)
	})

	t.Run("replace and disable", func(t *testing.T) {
		changes, err := parseConfigSetFlags(t, "--rid_isa_ttl=1h", "--disable=rid_sub")
		require.NoError(t, err)
		got, err := changes["rid"].apply("rid", current)
		require.NoError(t, err)
		require.Equal(t, &restapi.EvictionConfig{
			Interval: "1h0m0s",
			Entities: []restapi.EntityEviction{{Type: "isas", Ttl: "1h0m0s", Limit: new(int64(0))}},
		}, got, "the TTL and limit are replaced together, the disabled entity type removed")
	})
}
