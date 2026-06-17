package raftstore

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/interuss/dss/pkg/memstore"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/raftstore"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/dss/pkg/scd/repos"
	scdmemstore "github.com/interuss/dss/pkg/scd/store/memstore"
	"github.com/interuss/stacktrace"

	"go.uber.org/zap"
)

const storeID = "scd"

const (
	getOperationalIntent           raftstore.RequestType = "getOperationalIntent"
	deleteOperationalIntent        raftstore.RequestType = "deleteOperationalIntent"
	upsertOperationalIntent        raftstore.RequestType = "upsertOperationalIntent"
	searchOperationalIntents       raftstore.RequestType = "searchOperationalIntents"
	getDependentOperationalIntents raftstore.RequestType = "getDependentOperationalIntents"
	listExpiredOperationalIntents  raftstore.RequestType = "listExpiredOperationalIntents"
	countOperationalIntents        raftstore.RequestType = "countOperationalIntents"

	searchSubscriptions       raftstore.RequestType = "searchSubscriptions"
	getSubscription           raftstore.RequestType = "getSubscription"
	upsertSubscription        raftstore.RequestType = "upsertSubscription"
	deleteSubscription        raftstore.RequestType = "deleteSubscription"
	incrementNotificationIdxs raftstore.RequestType = "incrementNotificationIdxs"
	lockSubscriptionCells     raftstore.RequestType = "lockSubscriptionCells"
	listExpiredSubscriptions  raftstore.RequestType = "listExpiredSubscriptions"
	countSubscriptions        raftstore.RequestType = "countSubscriptions"

	getUSSAvailability    raftstore.RequestType = "getUSSAvailability"
	upsertUSSAvailability raftstore.RequestType = "upsertUSSAvailability"

	searchConstraints raftstore.RequestType = "searchConstraints"
	getConstraint     raftstore.RequestType = "getConstraint"
	upsertConstraint  raftstore.RequestType = "upsertConstraint"
	deleteConstraint  raftstore.RequestType = "deleteConstraint"
	countConstraints  raftstore.RequestType = "countConstraints"

	// Transactions
	DeleteOperationalIntentTransaction raftstore.RequestType = "deleteOperationalIntentTransaction"
	GetOperationalIntentTransaction    raftstore.RequestType = "getOperationalIntentTransaction"
	QueryOperationalIntentTransaction  raftstore.RequestType = "queryOperationalIntentTransaction"
	UpsertOperationalIntentTransaction raftstore.RequestType = "upsertOperationalIntentTransaction"

	DeleteSubscriptionTransaction raftstore.RequestType = "deleteSubscriptionTransaction"
	GetSubscriptionTransaction    raftstore.RequestType = "getSubscriptionTransaction"
	QuerySubscriptionTransaction  raftstore.RequestType = "querySubscriptionTransaction"
	UpsertSubscriptionTransaction raftstore.RequestType = "upsertSubscriptionTransaction"

	DeleteConstraintTransaction raftstore.RequestType = "deleteConstraintTransaction"
	GetConstraintTransaction    raftstore.RequestType = "getConstraintTransaction"
	QueryConstraintTransaction  raftstore.RequestType = "queryConstraintTransaction"
	UpsertConstraintTransaction raftstore.RequestType = "upsertConstraintTransaction"

	GetUSSAvailabilityTransaction raftstore.RequestType = "getUSSAvailabilityTransaction"
	SetUSSAvailabilityTransaction raftstore.RequestType = "setUSSAvailabilityTransaction"
)

var readOnlyRequests = []raftstore.RequestType{
	getOperationalIntent,
	searchOperationalIntents,
	getDependentOperationalIntents,
	listExpiredOperationalIntents,
	countOperationalIntents,

	searchSubscriptions,
	getSubscription,
	listExpiredSubscriptions,
	countSubscriptions,

	getUSSAvailability,

	searchConstraints,
	getConstraint,
	countConstraints,

	GetOperationalIntentTransaction,
	QueryOperationalIntentTransaction,

	GetSubscriptionTransaction,
	QuerySubscriptionTransaction,

	GetConstraintTransaction,
	QueryConstraintTransaction,

	GetUSSAvailabilityTransaction,
}

// repo is a full implementation of scd.repos.Repository for Raft-based storage.
type repo struct {
	consensus *consensus.Consensus

	memRepo  repos.Repository
	memStore *memstore.Store[repos.Repository]
}

func Init(ctx context.Context, logger *zap.Logger) (*raftstore.Store[repos.Repository], error) {
	memStore, err := scdmemstore.Init(ctx, logger)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to initialize scd memstore")
	}

	mem, err := memStore.Interact(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to obtain scd memstore repository")
	}

	r := &repo{memRepo: mem, memStore: memStore}

	store, consensus, err := raftstore.Init(ctx, logger, storeID, r)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to initialize scd raftstore")
	}

	r.consensus = consensus
	return store, err
}

func (r *repo) GetRepo() repos.Repository { return r }

func (r *repo) IsReadOnly(requestType raftstore.RequestType) bool {
	return slices.Contains(readOnlyRequests, requestType)
}

func (r *repo) GetSnapshot() ([]byte, error) {
	return nil, nil // TODO - implement
}

func (r *repo) RestoreFromSnapshot([]byte) error {
	return nil // TODO - implement
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
	case getUSSAvailability:
		var id dssmodels.Manager
		if err := json.Unmarshal(proposal.Value, &id); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", getUSSAvailability)
		}

		return r.memRepo.GetUssAvailability(ctx, id)

	case upsertUSSAvailability:
		var ussa scdmodels.UssAvailabilityStatus
		if err := json.Unmarshal(proposal.Value, &ussa); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", upsertUSSAvailability)
		}

		return r.memRepo.UpsertUssAvailability(ctx, &ussa)

	case searchConstraints:
		var v4d *dssmodels.Volume4D
		if err := json.Unmarshal(proposal.Value, &v4d); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", searchConstraints)
		}

		return r.memRepo.SearchConstraints(ctx, v4d)

	case getConstraint:
		var id dssmodels.ID
		if err := json.Unmarshal(proposal.Value, &id); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", getConstraint)
		}

		return r.memRepo.GetConstraint(ctx, id)

	case upsertConstraint:
		var constraint *scdmodels.Constraint
		if err := json.Unmarshal(proposal.Value, &constraint); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", upsertConstraint)
		}

		return r.memRepo.UpsertConstraint(ctx, constraint)

	case deleteConstraint:
		var id dssmodels.ID
		if err := json.Unmarshal(proposal.Value, &id); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", deleteConstraint)
		}

		return nil, r.memRepo.DeleteConstraint(ctx, id)

	case countConstraints:
		return r.memRepo.CountConstraints(ctx)

	case getOperationalIntent:
		var id dssmodels.ID
		if err := json.Unmarshal(proposal.Value, &id); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", getOperationalIntent)
		}

		return r.memRepo.GetOperationalIntent(ctx, id)

	case deleteOperationalIntent:
		var id dssmodels.ID
		if err := json.Unmarshal(proposal.Value, &id); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", deleteOperationalIntent)
		}

		return nil, r.memRepo.DeleteOperationalIntent(ctx, id)

	case upsertOperationalIntent:
		var operation *scdmodels.OperationalIntent
		if err := json.Unmarshal(proposal.Value, &operation); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", upsertOperationalIntent)
		}

		return r.memRepo.UpsertOperationalIntent(ctx, operation)

	case searchOperationalIntents:
		var v4d *dssmodels.Volume4D
		if err := json.Unmarshal(proposal.Value, &v4d); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", searchOperationalIntents)
		}

		return r.memRepo.SearchOperationalIntents(ctx, v4d)

	case getDependentOperationalIntents:
		var subscriptionID dssmodels.ID
		if err := json.Unmarshal(proposal.Value, &subscriptionID); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", getDependentOperationalIntents)
		}

		return r.memRepo.GetDependentOperationalIntents(ctx, subscriptionID)

	case listExpiredOperationalIntents:
		var threshold time.Time
		if err := json.Unmarshal(proposal.Value, &threshold); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", listExpiredOperationalIntents)
		}

		return r.memRepo.ListExpiredOperationalIntents(ctx, threshold)

	case countOperationalIntents:
		return r.memRepo.CountOperationalIntents(ctx)

	case searchSubscriptions:
		var v4d *dssmodels.Volume4D
		if err := json.Unmarshal(proposal.Value, &v4d); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", searchSubscriptions)
		}

		return r.memRepo.SearchSubscriptions(ctx, v4d)

	case getSubscription:
		var id dssmodels.ID
		if err := json.Unmarshal(proposal.Value, &id); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", getSubscription)
		}

		return r.memRepo.GetSubscription(ctx, id)

	case upsertSubscription:
		var sub *scdmodels.Subscription
		if err := json.Unmarshal(proposal.Value, &sub); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", upsertSubscription)
		}

		return r.memRepo.UpsertSubscription(ctx, sub)

	case deleteSubscription:
		var id dssmodels.ID
		if err := json.Unmarshal(proposal.Value, &id); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", deleteSubscription)
		}

		return nil, r.memRepo.DeleteSubscription(ctx, id)

	case incrementNotificationIdxs:
		var subscriptionIds []dssmodels.ID
		if err := json.Unmarshal(proposal.Value, &subscriptionIds); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", incrementNotificationIdxs)
		}

		return r.memRepo.IncrementNotificationIndices(ctx, subscriptionIds)

	case lockSubscriptionCells:
		var payload lockSubscriptionsOnCellsPayload
		if err := json.Unmarshal(proposal.Value, &payload); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", lockSubscriptionCells)
		}

		return nil, r.memRepo.LockSubscriptionsOnCells(ctx, payload.Cells, payload.SubscriptionIds, payload.StartTime, payload.EndTime)

	case listExpiredSubscriptions:
		var threshold time.Time
		if err := json.Unmarshal(proposal.Value, &threshold); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s proposal value", listExpiredSubscriptions)
		}

		return r.memRepo.ListExpiredSubscriptions(ctx, threshold)

	case countSubscriptions:
		return r.memRepo.CountSubscriptions(ctx)

	case DeleteOperationalIntentTransaction:
		return r.deleteOperationalIntentTransactionApplier(ctx, proposal)

	case GetOperationalIntentTransaction:
		return r.getOperationalIntentTransactionApplier(ctx, proposal)

	case QueryOperationalIntentTransaction:
		return r.queryOperationalIntentTransactionApplier(ctx, proposal)

	case UpsertOperationalIntentTransaction:
		return r.upsertOperationalIntentTransactionApplier(ctx, proposal)

	case DeleteConstraintTransaction:
		return r.deleteConstraintTransactionApplier(ctx, proposal)

	case GetConstraintTransaction:
		return r.getConstraintTransactionApplier(ctx, proposal)

	case QueryConstraintTransaction:
		return r.queryConstraintTransactionApplier(ctx, proposal)

	case UpsertConstraintTransaction:
		return r.upsertConstraintTransactionApplier(ctx, proposal)

	default:
		return nil, stacktrace.NewError("unknown request type: %q", proposal.RequestType)
	}
}
