package actions

import (
	"context"
	"encoding/json"

	ridv1 "github.com/interuss/dss/pkg/api/ridv1"
	ridv2 "github.com/interuss/dss/pkg/api/ridv2"
	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/rid/repos"
	dssstore "github.com/interuss/dss/pkg/store"
	"github.com/interuss/stacktrace"
)

func init() {
	Registry[ridv1.DeleteSubscriptionOperationID] = dssstore.OperationHandler[repos.Repository]{
		Encode:  EncodeRequest,
		Decode:  DecodeV1DeleteSubscription,
		Execute: ExecuteDeleteSubscription,
	}
	Registry[ridv2.DeleteSubscriptionOperationID] = dssstore.OperationHandler[repos.Repository]{
		Encode:  EncodeRequest,
		Decode:  DecodeV2DeleteSubscription,
		Execute: ExecuteDeleteSubscription,
	}
}

func EncodeRequest(request dssstore.OperationRequest) ([]byte, error) {
	return json.Marshal(request)
}

func DecodeV1DeleteSubscription(buf []byte) (dssstore.OperationRequest, error) {
	var req ridv1.DeleteSubscriptionRequest
	if err := json.Unmarshal(buf, &req); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", ridv1.DeleteSubscriptionOperationID)
	}
	return &req, nil
}

func DecodeV2DeleteSubscription(buf []byte) (dssstore.OperationRequest, error) {
	var req ridv2.DeleteSubscriptionRequest
	if err := json.Unmarshal(buf, &req); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", ridv2.DeleteSubscriptionOperationID)
	}
	return &req, nil
}

func ExecuteDeleteSubscription(ctx context.Context, repo repos.Repository, request dssstore.OperationRequest) (any, error) {
	var (
		rawID      string
		rawVersion string
		clientID   *string
	)

	switch req := request.(type) {
	case *ridv1.DeleteSubscriptionRequest:
		rawID, rawVersion, clientID = string(req.Id), req.Version, req.Auth.ClientID
	case *ridv2.DeleteSubscriptionRequest:
		rawID, rawVersion, clientID = string(req.Id), req.Version, req.Auth.ClientID
	default:
		return nil, stacktrace.NewError("unexpected request type %T for operation %q", request, ridv2.DeleteSubscriptionOperationID)
	}

	version, err := dssmodels.VersionFromString(rawVersion)
	if err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid version")
	}
	id, err := dssmodels.IDFromString(rawID)
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format")
	}
	owner := dssmodels.Owner(*clientID)

	old, err := repo.GetSubscription(ctx, id)
	switch {
	case err != nil:
		return nil, stacktrace.Propagate(err, "Error getting Subscription from repo")
	case old == nil:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "Subscription %s not found", id.String())
	case !version.Matches(old.Version):
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.VersionMismatch, "Subscription version %s is not current", version),
			"Subscription currently at version %s but client specified %s", old.Version, version)
	case old.Owner != owner:
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Subscription is owned by different client"),
			"Subscription owned by %s, but %s attempted to delete", old.Owner, owner)
	}

	ret, err := repo.DeleteSubscription(ctx, old)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error deleting Subscription from repo")
	}
	return ret, nil
}
