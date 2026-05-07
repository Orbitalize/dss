package server

import (
	"context"
	"time"

	"github.com/interuss/dss/pkg/api"
	restapi "github.com/interuss/dss/pkg/api/surveillancev0"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/dss/pkg/geo"
	dssmodels "github.com/interuss/dss/pkg/models"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	survmodels "github.com/interuss/dss/pkg/surveillance/models"
	"github.com/interuss/stacktrace"
	"github.com/pkg/errors"
)

// GetTrafficSurveilledArea returns a single TSA for a given ID.
func (s *Server) GetTrafficSurveilledArea(ctx context.Context, req *restapi.GetTrafficSurveilledAreaRequest,
) restapi.GetTrafficSurveilledAreaResponseSet {

	id, err := dssmodels.IDFromString(string(req.Id))
	if err != nil {
		return restapi.GetTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format"))}}
	}

	tsa, err := s.App.GetISA(ctx, id)
	if err != nil {
		return restapi.GetTrafficSurveilledAreaResponseSet{Response500: &api.InternalServerErrorBody{
			ErrorMessage: *dsserr.Handle(ctx, stacktrace.Propagate(err, "Could not get TSA from application layer"))}}
	}
	if tsa == nil {
		return restapi.GetTrafficSurveilledAreaResponseSet{Response404: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.NotFound, "TSA %s not found", req.Id))}}
	}
	return restapi.GetTrafficSurveilledAreaResponseSet{Response200: &restapi.GetTrafficSurveilledAreaResponse{
		SurveilledArea: *survmodels.ToTrafficSurveilledArea(tsa)}}
}

// CreateTrafficSurveilledArea creates an TSA
func (s *Server) CreateTrafficSurveilledArea(ctx context.Context, req *restapi.CreateTrafficSurveilledAreaRequest,
) restapi.CreateTrafficSurveilledAreaResponseSet {

	if req.Auth.ClientID == nil {
		return restapi.CreateTrafficSurveilledAreaResponseSet{Response403: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Missing owner"))}}
	}
	if req.BodyParseError != nil {
		return restapi.CreateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(req.BodyParseError, dsserr.BadRequest, "Malformed params"))}}
	}
	// TODO: put the validation logic in the models layer
	if req.Body.UssBaseUrl == "" {
		return restapi.CreateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing required USS base URL"))}}
	}
	extents, err := survmodels.FromVolume4D(&req.Body.Extents)
	if err != nil {
		return restapi.CreateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Error parsing Volume4D: %v", stacktrace.RootCause(err)))}}
	}
	id, err := dssmodels.IDFromString(string(req.Id))
	if err != nil {
		return restapi.CreateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format"))}}
	}

	if !s.AllowHTTPBaseUrls {
		err = ridmodels.ValidateURL(string(req.Body.UssBaseUrl))
		if err != nil {
			return restapi.CreateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
				Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Failed to validate base URL"))}}
		}
	}

	tsa := &survmodels.TrafficSurveilledArea{
		ID:     id,
		URL:    string(req.Body.UssBaseUrl),
		Owner:  dssmodels.Owner(*req.Auth.ClientID),
		Writer: s.Locality,
	}

	if err := tsa.SetExtents(extents); err != nil {
		return restapi.CreateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid extents"))}}
	}

	insertedTSA, subscribers, err := s.App.InsertISA(ctx, tsa)
	if err != nil {
		err = stacktrace.Propagate(err, "Could not insert TSA")
		errResp := &restapi.ErrorResponse{Message: dsserr.Handle(ctx, err)}
		switch stacktrace.GetCode(err) {
		case dsserr.AlreadyExists:
			return restapi.CreateTrafficSurveilledAreaResponseSet{Response409: errResp}
		case dsserr.BadRequest:
			return restapi.CreateTrafficSurveilledAreaResponseSet{Response400: errResp}
		default:
			return restapi.CreateTrafficSurveilledAreaResponseSet{Response500: &api.InternalServerErrorBody{
				ErrorMessage: *dsserr.Handle(ctx, stacktrace.Propagate(err, "Got an unexpected error"))}}
		}
	}

	apiSubscribers := survmodels.MakeSubscribersToNotify(subscribers)

	return restapi.CreateTrafficSurveilledAreaResponseSet{Response200: &restapi.PutTrafficSurveilledAreaResponse{
		SurveilledArea: *survmodels.ToTrafficSurveilledArea(insertedTSA),
		Subscribers:    &apiSubscribers,
	}}
}

// UpdateTrafficSurveilledArea updates an existing TSA.
func (s *Server) UpdateTrafficSurveilledArea(ctx context.Context, req *restapi.UpdateTrafficSurveilledAreaRequest,
) restapi.UpdateTrafficSurveilledAreaResponseSet {

	version, err := dssmodels.VersionFromString(req.Version)
	if err != nil {
		return restapi.UpdateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid version"))}}
	}

	if req.Auth.ClientID == nil {
		return restapi.UpdateTrafficSurveilledAreaResponseSet{Response403: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Missing owner"))}}
	}
	// TODO: put the validation logic in the models layer
	if req.BodyParseError != nil {
		return restapi.UpdateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(req.BodyParseError, dsserr.BadRequest, "Malformed params"))}}
	}
	if req.Body.UssBaseUrl == "" {
		return restapi.UpdateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing required USS base URL"))}}
	}
	extents, err := survmodels.FromVolume4D(&req.Body.Extents)
	if err != nil {
		return restapi.UpdateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Error parsing Volume4D: %v", stacktrace.RootCause(err)))}}
	}
	id, err := dssmodels.IDFromString(string(req.Id))
	if err != nil {
		return restapi.UpdateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format"))}}
	}

	tsa := &survmodels.TrafficSurveilledArea{
		ID:      id,
		URL:     string(req.Body.UssBaseUrl),
		Owner:   dssmodels.Owner(*req.Auth.ClientID),
		Version: version,
		Writer:  s.Locality,
	}

	if err := tsa.SetExtents(extents); err != nil {
		return restapi.UpdateTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid extents"))}}
	}

	insertedTSA, subscribers, err := s.App.UpdateISA(ctx, tsa)
	if err != nil {
		err = stacktrace.Propagate(err, "Could not update TSA")
		errResp := &restapi.ErrorResponse{Message: dsserr.Handle(ctx, err)}
		switch stacktrace.GetCode(err) {
		case dsserr.PermissionDenied:
			return restapi.UpdateTrafficSurveilledAreaResponseSet{Response403: errResp}
		case dsserr.VersionMismatch:
			return restapi.UpdateTrafficSurveilledAreaResponseSet{Response409: errResp}
		case dsserr.BadRequest, dsserr.NotFound:
			return restapi.UpdateTrafficSurveilledAreaResponseSet{Response400: errResp}
		default:
			return restapi.UpdateTrafficSurveilledAreaResponseSet{Response500: &api.InternalServerErrorBody{
				ErrorMessage: *dsserr.Handle(ctx, stacktrace.Propagate(err, "Got an unexpected error"))}}
		}
	}

	apiSubscribers := survmodels.MakeSubscribersToNotify(subscribers)

	return restapi.UpdateTrafficSurveilledAreaResponseSet{Response200: &restapi.PutTrafficSurveilledAreaResponse{
		SurveilledArea: *survmodels.ToTrafficSurveilledArea(insertedTSA),
		Subscribers:    &apiSubscribers,
	}}
}

// DeleteTrafficSurveilledArea deletes an existing TSA.
func (s *Server) DeleteTrafficSurveilledArea(ctx context.Context, req *restapi.DeleteTrafficSurveilledAreaRequest,
) restapi.DeleteTrafficSurveilledAreaResponseSet {

	if req.Auth.ClientID == nil {
		return restapi.DeleteTrafficSurveilledAreaResponseSet{Response403: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Missing owner"))}}
	}

	version, err := dssmodels.VersionFromString(req.Version)
	if err != nil {
		return restapi.DeleteTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid version"))}}
	}
	id, err := dssmodels.IDFromString(string(req.Id))
	if err != nil {
		return restapi.DeleteTrafficSurveilledAreaResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format"))}}
	}
	tsa, subscribers, err := s.App.DeleteISA(ctx, id, dssmodels.Owner(*req.Auth.ClientID), version)
	if err != nil {
		err = stacktrace.Propagate(err, "Could not delete TSA")
		errResp := &restapi.ErrorResponse{Message: dsserr.Handle(ctx, err)}
		switch stacktrace.GetCode(err) {
		case dsserr.PermissionDenied:
			return restapi.DeleteTrafficSurveilledAreaResponseSet{Response403: errResp}
		case dsserr.VersionMismatch:
			return restapi.DeleteTrafficSurveilledAreaResponseSet{Response409: errResp}
		case dsserr.NotFound:
			return restapi.DeleteTrafficSurveilledAreaResponseSet{Response404: errResp}
		default:
			return restapi.DeleteTrafficSurveilledAreaResponseSet{Response500: &api.InternalServerErrorBody{
				ErrorMessage: *dsserr.Handle(ctx, stacktrace.Propagate(err, "Got an unexpected error"))}}
		}
	}

	apiSubscribers := survmodels.MakeSubscribersToNotify(subscribers)

	return restapi.DeleteTrafficSurveilledAreaResponseSet{Response200: &restapi.DeleteTrafficSurveilledAreaResponse{
		SurveilledArea: *survmodels.ToTrafficSurveilledArea(tsa),
		Subscribers:    &apiSubscribers,
	}}
}

// SearchTrafficSurveilledAreas queries for all TSAs in the bounds.
func (s *Server) SearchTrafficSurveilledAreas(ctx context.Context, req *restapi.SearchTrafficSurveilledAreasRequest,
) restapi.SearchTrafficSurveilledAreasResponseSet {

	if req.Area == nil {
		return restapi.SearchTrafficSurveilledAreasResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing area"))}}
	}
	cu, err := geo.AreaToCellIDs(string(*req.Area))
	if err != nil {
		if errors.Is(err, geo.ErrAreaTooLarge) {
			return restapi.SearchTrafficSurveilledAreasResponseSet{Response413: &restapi.ErrorResponse{
				Message: dsserr.Handle(ctx, stacktrace.Propagate(err, "Invalid area"))}}
		}
		return restapi.SearchTrafficSurveilledAreasResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid area"))}}
	}

	var (
		earliest *time.Time
		latest   *time.Time
	)

	if req.EarliestTime != nil {
		ts, err := time.Parse(time.RFC3339Nano, *req.EarliestTime)
		if err != nil {
			return restapi.SearchTrafficSurveilledAreasResponseSet{Response400: &restapi.ErrorResponse{
				Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Unable to convert earliest timestamp"))}}
		}
		earliest = &ts
	}

	if req.LatestTime != nil {
		ts, err := time.Parse(time.RFC3339Nano, *req.LatestTime)
		if err != nil {
			return restapi.SearchTrafficSurveilledAreasResponseSet{Response400: &restapi.ErrorResponse{
				Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Unable to convert latest timestamp"))}}
		}
		latest = &ts
	}

	tsas, err := s.App.SearchISAs(ctx, cu, earliest, latest)
	if err != nil {
		err = stacktrace.Propagate(err, "Unable to search TSAs")
		if stacktrace.GetCode(err) == dsserr.BadRequest {
			return restapi.SearchTrafficSurveilledAreasResponseSet{Response400: &restapi.ErrorResponse{
				Message: dsserr.Handle(ctx, err)}}
		}
		return restapi.SearchTrafficSurveilledAreasResponseSet{Response500: &api.InternalServerErrorBody{
			ErrorMessage: *dsserr.Handle(ctx, stacktrace.Propagate(err, "Got an unexpected error"))}}
	}

	areas := make([]restapi.TrafficSurveilledArea, 0, len(tsas))
	for _, tsa := range tsas {
		areas = append(areas, *survmodels.ToTrafficSurveilledArea(tsa))
	}

	return restapi.SearchTrafficSurveilledAreasResponseSet{Response200: &restapi.SearchTrafficSurveilledAreasResponse{
		SurveilledAreas: &areas,
	}}
}
