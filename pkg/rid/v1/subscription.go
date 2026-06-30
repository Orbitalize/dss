package v1

import (
	"context"

	"github.com/interuss/dss/pkg/actions/ridv1"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/dss/pkg/geo"
	"github.com/interuss/dss/pkg/locality"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/rid"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	apiv1 "github.com/interuss/dss/pkg/rid/models/api/v1"
	"github.com/interuss/dss/pkg/rid/repos"
	"github.com/interuss/stacktrace"
	pkgerrors "github.com/pkg/errors"
)

func ExecuteGetSubscription(ctx context.Context, r repos.Repository, a *ridv1.GetSubscriptionAction) (any, error) {
	id, err := dssmodels.IDFromString(string(a.Id))
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format")
	}
	return r.GetSubscription(ctx, id)
}

func ExecuteSearchSubscriptions(ctx context.Context, r repos.Repository, a *ridv1.SearchSubscriptionsAction) (any, error) {
	if a.Auth.ClientID == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Missing owner")
	}
	if a.Area == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing area")
	}
	cu, err := geo.AreaToCellIDs(string(*a.Area))
	if err != nil {
		if pkgerrors.Is(err, geo.ErrAreaTooLarge) {
			return nil, stacktrace.PropagateWithCode(err, dsserr.AreaTooLarge, "Invalid area")
		}
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid area")
	}

	return r.SearchSubscriptionsByOwner(ctx, cu, dssmodels.Owner(*a.Auth.ClientID))
}

func ExecuteCreateSubscription(ctx context.Context, r repos.Repository, a *ridv1.CreateSubscriptionAction) (any, error) {
	if a.Auth.ClientID == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Missing owner")
	}
	if a.BodyParseError != nil {
		return nil, stacktrace.PropagateWithCode(a.BodyParseError, dsserr.BadRequest, "Malformed params")
	}
	if a.Body.Callbacks.IdentificationServiceAreaUrl == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing required callbacks")
	}
	if len(a.Body.Extents.SpatialVolume.Footprint.Vertices) == 0 {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing required extents")
	}
	extents, err := apiv1.FromVolume4D(&a.Body.Extents)
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Error parsing Volume4D: %v", stacktrace.RootCause(err))
	}
	id, err := dssmodels.IDFromString(string(a.Id))
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format")
	}

	if !rid.AllowHTTPBaseUrlsFromContext(ctx) {
		if err := ridmodels.ValidateURL(string(*a.Body.Callbacks.IdentificationServiceAreaUrl)); err != nil {
			return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Failed to validate IdentificationServiceAreaUrl")
		}
	}

	writer, err := locality.RequestLocalityFromContext(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get request locality")
	}

	sub := &ridmodels.Subscription{
		ID:     id,
		Owner:  dssmodels.Owner(*a.Auth.ClientID),
		URL:    string(*a.Body.Callbacks.IdentificationServiceAreaUrl),
		Writer: writer,
	}
	if err := sub.SetExtents(extents); err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid extents")
	}

	return rid.InsertSubscription(ctx, r, sub)
}

func ExecuteUpdateSubscription(ctx context.Context, r repos.Repository, a *ridv1.UpdateSubscriptionAction) (any, error) {
	version, err := dssmodels.VersionFromString(a.Version)
	if err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid version")
	}
	id, err := dssmodels.IDFromString(string(a.Id))
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format")
	}
	if a.Auth.ClientID == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Missing owner")
	}
	if a.BodyParseError != nil {
		return nil, stacktrace.PropagateWithCode(a.BodyParseError, dsserr.BadRequest, "Malformed params")
	}
	if a.Body.Callbacks.IdentificationServiceAreaUrl == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing required callbacks")
	}
	if len(a.Body.Extents.SpatialVolume.Footprint.Vertices) == 0 {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing required extents")
	}
	extents, err := apiv1.FromVolume4D(&a.Body.Extents)
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Error parsing Volume4D: %v", stacktrace.RootCause(err))
	}

	writer, err := locality.RequestLocalityFromContext(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get request locality")
	}

	sub := &ridmodels.Subscription{
		ID:      id,
		Owner:   dssmodels.Owner(*a.Auth.ClientID),
		URL:     string(*a.Body.Callbacks.IdentificationServiceAreaUrl),
		Version: version,
		Writer:  writer,
	}
	if err := sub.SetExtents(extents); err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid extents")
	}

	return rid.UpdateSubscription(ctx, r, sub)
}

func ExecuteDeleteSubscription(ctx context.Context, r repos.Repository, a *ridv1.DeleteSubscriptionAction) (any, error) {
	if a.Auth.ClientID == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Missing owner")
	}
	version, err := dssmodels.VersionFromString(a.Version)
	if err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid version")
	}
	id, err := dssmodels.IDFromString(string(a.Id))
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format")
	}

	return rid.DeleteSubscription(ctx, r, id, dssmodels.Owner(*a.Auth.ClientID), version)
}
