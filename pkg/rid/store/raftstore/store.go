package raftstore

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/golang/geo/s2"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/dss/pkg/memstore"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/raftstore"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/dss/pkg/rid/repos"
	ridmemstore "github.com/interuss/dss/pkg/rid/store/memstore"
	"github.com/interuss/stacktrace"
	"go.uber.org/zap"
)

const storeID = "rid"

const (
	getISA          raftstore.RequestType = "getISA"
	deleteISA       raftstore.RequestType = "deleteISA"
	insertISA       raftstore.RequestType = "insertISA"
	updateISA       raftstore.RequestType = "updateISA"
	searchISAs      raftstore.RequestType = "searchISAs"
	listExpiredISAs raftstore.RequestType = "listExpiredISAs"
	countISAs       raftstore.RequestType = "countISAs"

	DeleteISATransaction raftstore.RequestType = "deleteISATransaction"
	InsertISATransaction raftstore.RequestType = "insertISATransaction"
	UpdateISATransaction raftstore.RequestType = "updateISATransaction"

	getSubscription                    raftstore.RequestType = "getSubscription"
	deleteSubscription                 raftstore.RequestType = "deleteSubscription"
	insertSubscription                 raftstore.RequestType = "insertSubscription"
	updateSubscription                 raftstore.RequestType = "updateSubscription"
	searchSubscriptions                raftstore.RequestType = "searchSubscriptions"
	searchSubscriptionsByOwner         raftstore.RequestType = "searchSubscriptionsByOwner"
	updateNotificationIdxsInCells      raftstore.RequestType = "updateNotificationIdxsInCells"
	maxSubscriptionCountInCellsByOwner raftstore.RequestType = "maxSubscriptionCountInCellsByOwner"
	listExpiredSubscriptions           raftstore.RequestType = "listExpiredSubscriptions"
	countSubscriptions                 raftstore.RequestType = "countSubscriptions"

	DeleteSubscriptionTransaction raftstore.RequestType = "deleteSubscriptionTransaction"
	InsertSubscriptionTransaction raftstore.RequestType = "insertSubscriptionTransaction"
	UpdateSubscriptionTransaction raftstore.RequestType = "updateSubscriptionTransaction"
)

var readOnlyRequests = []raftstore.RequestType{
	getISA,
	searchISAs,
	listExpiredISAs,
	countISAs,
	getSubscription,
	searchSubscriptions,
	searchSubscriptionsByOwner,
	maxSubscriptionCountInCellsByOwner,
	listExpiredSubscriptions,
	countSubscriptions,
}

// repo is a full implementation of rid.repos.Repository for Raft-based storage.
type repo struct {
	consensus *consensus.Consensus

	memRepo  repos.Repository
	memStore *memstore.Store[repos.Repository]
}

func Init(ctx context.Context, logger *zap.Logger) (*raftstore.Store[repos.Repository], error) {
	memStore, err := ridmemstore.Init(ctx, logger)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to initialize rid memstore")
	}

	mem, err := memStore.Interact(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to obtain rid memstore repository")
	}

	r := &repo{memRepo: mem, memStore: memStore}

	store, consensus, err := raftstore.Init(ctx, logger, storeID, r)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to initialize rid raftstore")
	}

	r.consensus = consensus
	return store, err
}

func (r *repo) GetRepo() repos.Repository { return r }

func (r *repo) IsReadOnly(requestType raftstore.RequestType) bool {
	return slices.Contains(readOnlyRequests, requestType)
}

func (r *repo) GetSnapshot() ([]byte, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "not implemented yet")
}

func (r *repo) RestoreFromSnapshot([]byte) error {
	return stacktrace.NewErrorWithCode(dsserr.NotImplemented, "not implemented yet")
}

func (r *repo) propose(ctx context.Context, requestType raftstore.RequestType, payload any) (any, error) {
	proposal, err := consensus.NewProposal(ctx, storeID, string(requestType), payload, r.IsReadOnly(requestType), nil)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to create %s proposal", requestType)
	}
	return r.consensus.ProposeValue(ctx, proposal)
}

func (r *repo) Apply(ctx context.Context, proposal consensus.Proposal) (any, error) {
	switch raftstore.RequestType(proposal.RequestType) {

	// ISAs

	case getISA:
		var p getISAPayload
		if err := json.Unmarshal(proposal.Value, &p); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", getISA)
		}
		return r.memRepo.GetISA(ctx, p.ID, p.ForUpdate)

	case deleteISA:
		var isa ridmodels.IdentificationServiceArea
		if err := json.Unmarshal(proposal.Value, &isa); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", deleteISA)
		}
		return r.memRepo.DeleteISA(ctx, &isa)

	case insertISA:
		var isa ridmodels.IdentificationServiceArea
		if err := json.Unmarshal(proposal.Value, &isa); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", insertISA)
		}
		return r.memRepo.InsertISA(ctx, &isa)

	case updateISA:
		var isa ridmodels.IdentificationServiceArea
		if err := json.Unmarshal(proposal.Value, &isa); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", updateISA)
		}
		return r.memRepo.UpdateISA(ctx, &isa)

	case searchISAs:
		var p searchISAsPayload
		if err := json.Unmarshal(proposal.Value, &p); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", searchISAs)
		}
		return r.memRepo.SearchISAs(ctx, p.Cells, p.Earliest, p.Latest)

	case listExpiredISAs:
		var p listExpiredISAsPayload
		if err := json.Unmarshal(proposal.Value, &p); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", listExpiredISAs)
		}
		return r.memRepo.ListExpiredISAs(ctx, p.Writer, p.Threshold)

	case countISAs:
		return r.memRepo.CountISAs(ctx)

	case DeleteISATransaction:
		return r.deleteISATransactionApplier(ctx, proposal)

	case InsertISATransaction:
		return r.insertISATransactionApplier(ctx, proposal)

	case UpdateISATransaction:
		return r.updateISATransactionApplier(ctx, proposal)

	// Subscriptions

	case getSubscription:
		var id dssmodels.ID
		if err := json.Unmarshal(proposal.Value, &id); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", getSubscription)
		}
		return r.memRepo.GetSubscription(ctx, id)

	case deleteSubscription:
		var sub ridmodels.Subscription
		if err := json.Unmarshal(proposal.Value, &sub); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", deleteSubscription)
		}
		return r.memRepo.DeleteSubscription(ctx, &sub)

	case insertSubscription:
		var sub ridmodels.Subscription
		if err := json.Unmarshal(proposal.Value, &sub); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", insertSubscription)
		}
		return r.memRepo.InsertSubscription(ctx, &sub)

	case updateSubscription:
		var sub ridmodels.Subscription
		if err := json.Unmarshal(proposal.Value, &sub); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", updateSubscription)
		}
		return r.memRepo.UpdateSubscription(ctx, &sub)

	case searchSubscriptions:
		var cells s2.CellUnion
		if err := json.Unmarshal(proposal.Value, &cells); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", searchSubscriptions)
		}
		return r.memRepo.SearchSubscriptions(ctx, cells)

	case searchSubscriptionsByOwner:
		var p searchSubscriptionsByOwnerPayload
		if err := json.Unmarshal(proposal.Value, &p); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", searchSubscriptionsByOwner)
		}
		return r.memRepo.SearchSubscriptionsByOwner(ctx, p.Cells, p.Owner)

	case updateNotificationIdxsInCells:
		var cells s2.CellUnion
		if err := json.Unmarshal(proposal.Value, &cells); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", updateNotificationIdxsInCells)
		}
		return r.memRepo.UpdateNotificationIdxsInCells(ctx, cells)

	case maxSubscriptionCountInCellsByOwner:
		var p maxSubscriptionCountInCellsByOwnerPayload
		if err := json.Unmarshal(proposal.Value, &p); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", maxSubscriptionCountInCellsByOwner)
		}
		return r.memRepo.MaxSubscriptionCountInCellsByOwner(ctx, p.Cells, p.Owner)

	case listExpiredSubscriptions:
		var p listExpiredSubscriptionsPayload
		if err := json.Unmarshal(proposal.Value, &p); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", listExpiredSubscriptions)
		}
		return r.memRepo.ListExpiredSubscriptions(ctx, p.Writer, p.Threshold)

	case countSubscriptions:
		return r.memRepo.CountSubscriptions(ctx)

	case DeleteSubscriptionTransaction:
		return r.deleteSubscriptionTransactionApplier(ctx, proposal)

	case InsertSubscriptionTransaction:
		return r.insertSubscriptionTransactionApplier(ctx, proposal)

	case UpdateSubscriptionTransaction:
		return r.updateSubscriptionTransactionApplier(ctx, proposal)

	default:
		return nil, stacktrace.NewError("unknown request type: %q", proposal.RequestType)
	}
}
