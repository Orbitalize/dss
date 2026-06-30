package v2

import (
	"context"

	"github.com/interuss/dss/pkg/actions/ridv2"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/dss/pkg/geo"
	"github.com/interuss/dss/pkg/locality"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/rid"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	apiv2 "github.com/interuss/dss/pkg/rid/models/api/v2"
	"github.com/interuss/dss/pkg/rid/repos"
	"github.com/interuss/stacktrace"
	pkgerrors "github.com/pkg/errors"
)

func ExecuteGetSubscription(ctx context.Context, r repos.Repository, a *ridv2.GetSubscriptionAction) (any, error) {
	id, err := dssmodels.IDFromString(string(a.Id))
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format")
	}
	return r.GetSubscription(ctx, id)
}

func ExecuteSearchSubscriptions(ctx context.Context, r repos.Repository, a *ridv2.SearchSubscriptionsAction) (any, error) {
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

func ExecuteCreateSubscription(ctx context.Context, r repos.Repository, a *ridv2.CreateSubscriptionAction) (any, error) {
	if a.Auth.ClientID == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Missing owner")
	}
	if a.BodyParseError != nil {
		return nil, stacktrace.PropagateWithCode(a.BodyParseError, dsserr.BadRequest, "Malformed params")
	}
	if a.Body.UssBaseUrl == "" {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing required USS base URL")
	}
	extents, err := apiv2.FromVolume4D(&a.Body.Extents)
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Error parsing Volume4D: %v", stacktrace.RootCause(err))
	}
	id, err := dssmodels.IDFromString(string(a.Id))
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format")
	}

	if !rid.AllowHTTPBaseUrlsFromContext(ctx) {
		if err := ridmodels.ValidateURL(string(a.Body.UssBaseUrl)); err != nil {
			return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Failed to validate UssBaseUrl")
		}
	}

	writer, err := locality.RequestLocalityFromContext(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get request locality")
	}

	sub := &ridmodels.Subscription{
		ID:     id,
		Owner:  dssmodels.Owner(*a.Auth.ClientID),
		URL:    string(a.Body.UssBaseUrl),
		Writer: writer,
	}
	if err := sub.SetExtents(extents); err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid extents")
	}

	return rid.InsertSubscription(ctx, r, sub)
}

func ExecuteUpdateSubscription(ctx context.Context, r repos.Repository, a *ridv2.UpdateSubscriptionAction) (any, error) {
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
	if a.Body.UssBaseUrl == "" {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing required USS base URL")
	}
	extents, err := apiv2.FromVolume4D(&a.Body.Extents)
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
		URL:     string(a.Body.UssBaseUrl),
		Version: version,
		Writer:  writer,
	}
	if err := sub.SetExtents(extents); err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid extents")
	}

	return rid.UpdateSubscription(ctx, r, sub)
}

func ExecuteDeleteSubscription(ctx context.Context, r repos.Repository, a *ridv2.DeleteSubscriptionAction) (any, error) {
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
