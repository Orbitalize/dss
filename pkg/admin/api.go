// Package admin defines the admin API of a DSS instance, used by dssctl, such as the management of the membership of its Raft groups.
// It only holds the contract between the DSS instance (e.g. raftstore.RegisterAdminHandlers) and dssctl (see the client package):
// it must not depend on the implementation of the stores, so that dssctl does not depend on them nor on their flags.
// The API is not yet authenticated (TODO).
package admin

import (
	"context"
	"fmt"
)

// RaftGroup manages the membership of a Raft group.
// It is implemented by raftstore.Store within the DSS instance and by the client package within dssctl.
type RaftGroup interface {
	// Members returns the membership of the group as known by the DSS instance.
	Members(ctx context.Context) (Membership, error)
	// AddNode adds nodeID, reachable by the other members at peerURL, to the group.
	AddNode(ctx context.Context, nodeID uint64, peerURL string) error
	// RemoveNode removes nodeID from the group. Its ID is retired and can never be reused.
	RemoveNode(ctx context.Context, nodeID uint64) error
}

// Membership describes the membership of a Raft group, as known by a node.
type Membership struct {
	// NodeID is the ID of the node reporting the membership.
	NodeID uint64 `json:"node_id"`
	// LeaderID is the ID of the current leader of the group, 0 if unknown.
	LeaderID uint64   `json:"leader_id"`
	Voters   []uint64 `json:"voters"`
	// Members maps the ID of every member of the group to its peer URL.
	Members    map[uint64]string `json:"members"`
	RemovedIDs []uint64          `json:"removed_ids"`
}

// MembersPath lists the members of a Raft group (GET) or adds one (POST).
func MembersPath(group string) string {
	return fmt.Sprintf("/raft/%s/members", group)
}

// MemberPath removes a member of a Raft group (DELETE).
func MemberPath(group string, nodeID uint64) string {
	return fmt.Sprintf("%s/%d", MembersPath(group), nodeID)
}

// AddNodeRequest is the body of a request adding a node to a Raft group.
type AddNodeRequest struct {
	NodeID uint64 `json:"node_id"`
	// URL is the URL at which the other members of the group reach the node.
	URL string `json:"url"`
}

// ErrorResponse is returned when a request fails.
type ErrorResponse struct {
	Error string `json:"error"`
}
