package consensus

import (
	"bytes"
	"encoding/gob"
	"maps"
	"slices"
	"sync"

	"github.com/interuss/stacktrace"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
)

// minVoters is the minimum number of voters a Raft group must keep.
// Below it, the group can no longer tolerate the loss of a single node without losing its quorum.
const minVoters = 3

// ErrConfChangeRejected is returned when a configuration change is refused because it would lead to an invalid membership.
var ErrConfChangeRejected = stacktrace.NewError("configuration change rejected")

type membership struct {
	sync.RWMutex

	confState raftpb.ConfState
	// members maps the ID of every member of the group to its peer URL.
	members map[uint64]string
	// removedIDs are the IDs of the nodes removed from the group
	// removed IDs can never be reused.
	removedIDs []uint64
}

func (m *membership) setConfState(confState raftpb.ConfState) {
	m.Lock()
	defer m.Unlock()
	m.confState = confState
}

// addNode records nodeID as a member reachable at peerURL.
func (m *membership) addNode(nodeID uint64, peerURL string) {
	m.Lock()
	defer m.Unlock()
	m.members[nodeID] = peerURL
}

// removeNode removes nodeID from the members and retires its ID.
func (m *membership) removeNode(nodeID uint64) {
	m.Lock()
	defer m.Unlock()
	delete(m.members, nodeID)
	m.removedIDs = append(m.removedIDs, nodeID)
}

// validateConfChange checks that applying change keeps the membership valid.
// It must be deterministic as it is evaluated by every node when the change is applied.
func (m *membership) validateConfChange(change raftpb.ConfChangeSingle, peerURL string) error {
	m.RLock()
	defer m.RUnlock()

	if change.NodeID == raft.None {
		return stacktrace.Propagate(ErrConfChangeRejected, "node ID %d is invalid", change.NodeID)
	}

	if slices.Contains(m.removedIDs, change.NodeID) {
		return stacktrace.Propagate(ErrConfChangeRejected, "node ID %d was removed from the group and cannot be reused", change.NodeID)
	}

	voters := m.confState.Voters
	isVoter := slices.Contains(voters, change.NodeID)
	switch change.Type {
	case raftpb.ConfChangeAddNode:
		if isVoter {
			return stacktrace.Propagate(ErrConfChangeRejected, "node %d is already a member of the group", change.NodeID)
		}
		if _, err := types.NewURLs([]string{peerURL}); err != nil {
			return stacktrace.Propagate(ErrConfChangeRejected, "invalid URL %q for node %d: %v", peerURL, change.NodeID, err)
		}

	case raftpb.ConfChangeRemoveNode:
		if !isVoter {
			return stacktrace.Propagate(ErrConfChangeRejected, "node %d is not a member of the group", change.NodeID)
		}
		if len(voters)-1 < minVoters {
			return stacktrace.Propagate(ErrConfChangeRejected, "removing node %d would leave %d voters, below the minimum of %d", change.NodeID, len(voters)-1, minVoters)
		}

	default:
		return stacktrace.Propagate(ErrConfChangeRejected, "unsupported config change type %s", change.Type)
	}

	return nil
}

// snapshot wraps the application snapshot with the membership state that the raft library does not track itself,
// so that a node catching up via a snapshot fully recovers it, including which IDs must never be reused.
type snapshot struct {
	// Members maps the ID of every member of the group to its peer URL.
	Members map[uint64]string
	// RemovedIDs are the IDs of the nodes permanently removed from the group.
	RemovedIDs []uint64
	// AppData is the application snapshot data.
	AppData []byte
}

// encodeSnapshot encodes the application snapshot with the current membership state into the snapshot data.
// TODO: Use protobuf instead of gob as a follow-up to https://github.com/interuss/dss/pull/1765
func (c *Consensus) encodeSnapshot() ([]byte, error) {
	appData, err := c.provider()
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to get application snapshot data")
	}

	c.membership.RLock()
	env := snapshot{
		Members:    maps.Clone(c.membership.members),
		RemovedIDs: slices.Clone(c.membership.removedIDs),
		AppData:    appData,
	}
	c.membership.RUnlock()

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(env); err != nil {
		return nil, stacktrace.Propagate(err, "failed to encode snapshot envelope")
	}
	return buf.Bytes(), nil
}

// decodeSnapshot decodes the snapshot data produced by encodeSnapshot.
func decodeSnapshot(data []byte) (snapshot, error) {
	var env snapshot
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&env); err != nil {
		return snapshot{}, stacktrace.Propagate(err, "failed to decode snapshot envelope")
	}
	return env, nil
}

// restoreMembership replaces the membership state and updates the transport peers accordingly.
func (c *Consensus) restoreMembership(members map[uint64]string, removedIDs []uint64) {
	c.membership.Lock()
	c.membership.members = maps.Clone(members)
	if c.membership.members == nil {
		c.membership.members = map[uint64]string{}
	}
	c.membership.removedIDs = slices.Clone(removedIDs)
	c.membership.Unlock()

	for _, id := range removedIDs {
		c.transport.RemovePeer(types.ID(id))
	}
	for id, peerURL := range members {
		if id != c.nodeID && peerURL != "" {
			c.transport.AddPeer(types.ID(id), []string{peerURL})
		}
	}
}
