// This file is auto-generated; do not change as any changes will be overwritten
package adminv1

type ErrorResponse struct {
	// Human-readable message indicating what error occurred and/or why.
	Message *string `json:"message,omitempty"`
}

// Name of a Raft group of a DSS node, one per service (e.g., `aux`, `rid`, `scd`).
type RaftGroupName string

// A DSS node in a Raft group.
type RaftNode struct {
	// Identifier for the node within the Raft group.
	Id string `json:"id"`

	// URL at which the other members of the Raft group send Raft messages to the node. Served by the Raft communication listener. It identifies the DSS node within the cluster and across all Raft groups.
	Url string `json:"url"`

	// State of the node in the Raft group.
	//
	// * `waiting`: Node is not yet a member of the group. It waits to be added to a running group or to create a new one.
	//
	// * `member`: Node is a member of the group.
	//
	// * `removed`: Node was removed from the group. Its ID can never be added to the group again.
	State string `json:"state"`
}

// Request to add another DSS node to a Raft group. This request is sent to a member of the group, which reads the ID of the node to add from the node itself, at its URL.
type CreateRaftGroupMemberParameters struct {
	// URL at which the members of the Raft group send Raft messages to the node to add.
	Url string `json:"url"`
}

// Request to create a Raft group with this DSS node as its only member. Used for bootstrapping a new Raft group, which is then joined by the other nodes one at a time.
type CreateRaftGroupParameters struct {
	GroupName RaftGroupName `json:"group_name"`
}

// A Raft group, as known by this DSS node.
type RaftGroup struct {
	GroupName RaftGroupName `json:"group_name"`

	// Identifier for the Raft group within the cluster. Identical on all the members of the group.
	ClusterId string `json:"cluster_id"`

	// Identifier for the current leader of the Raft group. Empty if unknown.
	LeaderId string `json:"leader_id"`

	// Nodes that are or were members of the Raft group.
	Nodes []RaftNode `json:"nodes"`
}

// Eviction parameters of an entity type of a Raft group.
type EntityEviction struct {
	// Entity type, unique within a request: `isas` or `subscriptions` for the `rid` group, `operational_intents` or `subscriptions` for the `scd` group.
	Type string `json:"type"`

	// Duration, as a Go duration string, after their end at which entities are expired. Must be positive.
	Ttl string `json:"ttl"`

	// Maximum number of entities evicted by a cleanup job, 0 for unlimited.
	Limit *int64 `json:"limit,omitempty"`
}

// Eviction settings of a Raft group, used by the cleanup jobs scheduled by its leader.
type EvictionConfig struct {
	// Duration, as a Go duration string, between two scheduled cleanup jobs. Must be positive.
	Interval string `json:"interval"`

	// Entity types evicted by the scheduled cleanup jobs. The entity types not listed are not evicted.
	Entities []EntityEviction `json:"entities"`
}

// Eviction settings of a Raft group, as stored in its replicated state. `config` is absent if they were never set, in which case nothing is evicted by the scheduled cleanup jobs.
type EvictionConfigResponse struct {
	Config *EvictionConfig `json:"config,omitempty"`
}

// Parameters of a cleanup job run on demand, independently of the eviction settings.
type EvictionRunParameters struct {
	// Entity types to evict. The entity types not listed are skipped.
	Entities []EntityEviction `json:"entities"`

	// Whether to delete the expired entities. When false, they are only counted, and the limits must not be set.
	Delete *bool `json:"delete,omitempty"`
}

// Number of entities of an entity type evicted, or that would be evicted, by a cleanup job.
type EntityEvictionCount struct {
	// Entity type.
	Type string `json:"type"`

	// Number of entities deleted, or, when the job only counts, number of expired entities regardless of the limit.
	Count int64 `json:"count"`
}

// Result of a cleanup job run on demand.
type EvictionRunResult struct {
	// Number of entities per entity type of the job, in the order of the request.
	Counts []EntityEvictionCount `json:"counts"`
}
