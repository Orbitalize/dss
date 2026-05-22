package consensus

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/interuss/dss/pkg/timestamp"
	"github.com/interuss/stacktrace"
)

type EntryCommit struct {
	Prop Proposal
	Done chan ProposalResult

	SnapshotData []byte
}

type Proposal struct {
	ID          string            `json:"id"`
	DBName      string            `json:"dbname"`
	Timestamp   time.Time         `json:"timestamp"`
	RequestType string            `json:"request_type"`
	Value       []byte            `json:"value"`
	ReadOnly    bool              `json:"read_only"`
	Parameters  map[string][]byte `json:"parameters,omitempty"`
}

type ProposalResult struct {
	Result any
	Error  error
}

func NewProposal(ctx context.Context, dbname string, requestType string, payload any, readOnly bool, parameters map[string][]byte) (Proposal, error) {
	proposalTimestamp := timestamp.NowFromContext(ctx)
	if proposalTimestamp.IsZero() {
		proposalTimestamp = time.Now().UTC()
	}

	value, err := json.Marshal(payload)
	if err != nil {
		return Proposal{}, stacktrace.Propagate(err, "failed to serialize proposal payload")
	}

	return Proposal{
		ID:          uuid.NewString(),
		DBName:      dbname,
		Timestamp:   proposalTimestamp,
		RequestType: requestType,
		Value:       value,
		ReadOnly:    readOnly,
		Parameters:  parameters,
	}, nil
}

type proposalsTracker struct {
	sync.Mutex
	pending map[string]chan ProposalResult
}

func newProposalsTracker() *proposalsTracker {
	return &proposalsTracker{
		pending: make(map[string]chan ProposalResult),
	}
}

func (p *proposalsTracker) isPending(id string) bool {
	p.Lock()
	defer p.Unlock()

	_, ok := p.pending[id]
	return ok
}

func (p *proposalsTracker) track(id string) chan ProposalResult {
	p.Lock()
	defer p.Unlock()

	applied := make(chan ProposalResult, 1)
	p.pending[id] = applied
	return applied
}

func (p *proposalsTracker) untrack(id string, result ProposalResult) {
	p.Lock()
	defer p.Unlock()

	applied := p.pending[id]
	applied <- result
	delete(p.pending, id)
}
