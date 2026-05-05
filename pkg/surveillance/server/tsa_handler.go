package server

import (
	"context"

	"github.com/interuss/dss/pkg/api"
	restapi "github.com/interuss/dss/pkg/api/surveillancev0"
)

// GetTrafficSurveilledArea returns a single TSA for a given ID.
func (s *Server) GetTrafficSurveilledArea(ctx context.Context, req *restapi.GetTrafficSurveilledAreaRequest,
) restapi.GetTrafficSurveilledAreaResponseSet {
	return restapi.GetTrafficSurveilledAreaResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}

// CreateTrafficSurveilledArea creates an TSA
func (s *Server) CreateTrafficSurveilledArea(ctx context.Context, req *restapi.CreateTrafficSurveilledAreaRequest,
) restapi.CreateTrafficSurveilledAreaResponseSet {
	return restapi.CreateTrafficSurveilledAreaResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}

// UpdateTrafficSurveilledArea updates an existing TSA.
func (s *Server) UpdateTrafficSurveilledArea(ctx context.Context, req *restapi.UpdateTrafficSurveilledAreaRequest,
) restapi.UpdateTrafficSurveilledAreaResponseSet {
	return restapi.UpdateTrafficSurveilledAreaResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}

// DeleteTrafficSurveilledArea deletes an existing TSA.
func (s *Server) DeleteTrafficSurveilledArea(ctx context.Context, req *restapi.DeleteTrafficSurveilledAreaRequest,
) restapi.DeleteTrafficSurveilledAreaResponseSet {
	return restapi.DeleteTrafficSurveilledAreaResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}

// SearchTrafficSurveilledAreas queries for all TSAs in the bounds.
func (s *Server) SearchTrafficSurveilledAreas(ctx context.Context, req *restapi.SearchTrafficSurveilledAreasRequest,
) restapi.SearchTrafficSurveilledAreasResponseSet {
	return restapi.SearchTrafficSurveilledAreasResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}
