package server

import (
	"context"
	"slices"
	"time"

	"github.com/interuss/dss/pkg/api"
	restapi "github.com/interuss/dss/pkg/api/adminv1"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/stacktrace"
	"go.uber.org/zap"
)

func (s *Server) GetEvictionConfig(ctx context.Context, req *restapi.GetEvictionConfigRequest) restapi.GetEvictionConfigResponseSet {
	evictor, err := s.evictor(req.GroupName)
	if err != nil {
		return restapi.GetEvictionConfigResponseSet{Response404: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, err)}}
	}

	config, err := evictor.GetEvictionConfig(ctx)
	if err != nil {
		return restapi.GetEvictionConfigResponseSet{Response500: &api.InternalServerErrorBody{
			ErrorMessage: *dsserr.Handle(ctx, stacktrace.Propagate(err, "Could not get the eviction config of the %s raft group", req.GroupName))}}
	}
	result := &restapi.EvictionConfigResponse{}
	if config != nil {
		result.Config = toEvictionConfig(evictor.EntityTypes(), *config)
	}
	return restapi.GetEvictionConfigResponseSet{Response200: result}
}

func (s *Server) PutEvictionConfig(ctx context.Context, req *restapi.PutEvictionConfigRequest) restapi.PutEvictionConfigResponseSet {
	evictor, err := s.evictor(req.GroupName)
	if err != nil {
		return restapi.PutEvictionConfigResponseSet{Response404: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, err)}}
	}
	if req.BodyParseError != nil {
		return restapi.PutEvictionConfigResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(req.BodyParseError, dsserr.BadRequest, "Malformed params"))}}
	}
	config, err := fromEvictionConfig(evictor.EntityTypes(), req.Body)
	if err != nil {
		return restapi.PutEvictionConfigResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, err)}}
	}

	stored, err := evictor.SetEvictionConfig(ctx, config)
	if err != nil {
		err = stacktrace.Propagate(err, "Could not set the eviction config of the %s raft group", req.GroupName)
		switch stacktrace.GetCode(err) {
		case dsserr.BadRequest:
			return restapi.PutEvictionConfigResponseSet{Response400: &restapi.ErrorResponse{Message: dsserr.Handle(ctx, err)}}
		default:
			return restapi.PutEvictionConfigResponseSet{Response500: &api.InternalServerErrorBody{
				ErrorMessage: *dsserr.Handle(ctx, stacktrace.Propagate(err, "Got an unexpected error"))}}
		}
	}

	s.Logger.Info("set eviction config", zap.String("group", string(req.GroupName)), zap.Duration("interval", stored.Interval))
	return restapi.PutEvictionConfigResponseSet{Response200: toEvictionConfig(evictor.EntityTypes(), *stored)}
}

func (s *Server) CreateEvictionRun(ctx context.Context, req *restapi.CreateEvictionRunRequest) restapi.CreateEvictionRunResponseSet {
	evictor, err := s.evictor(req.GroupName)
	if err != nil {
		return restapi.CreateEvictionRunResponseSet{Response404: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, err)}}
	}
	if req.BodyParseError != nil {
		return restapi.CreateEvictionRunResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, stacktrace.PropagateWithCode(req.BodyParseError, dsserr.BadRequest, "Malformed params"))}}
	}
	run, order, err := fromEvictionRunParameters(evictor.EntityTypes(), req.Body)
	if err != nil {
		return restapi.CreateEvictionRunResponseSet{Response400: &restapi.ErrorResponse{
			Message: dsserr.Handle(ctx, err)}}
	}

	counts, err := evictor.RunEviction(ctx, run)
	if err != nil {
		err = stacktrace.Propagate(err, "Could not run a cleanup job on the %s raft group", req.GroupName)
		switch stacktrace.GetCode(err) {
		case dsserr.BadRequest:
			return restapi.CreateEvictionRunResponseSet{Response400: &restapi.ErrorResponse{Message: dsserr.Handle(ctx, err)}}
		default:
			return restapi.CreateEvictionRunResponseSet{Response500: &api.InternalServerErrorBody{
				ErrorMessage: *dsserr.Handle(ctx, stacktrace.Propagate(err, "Got an unexpected error"))}}
		}
	}

	result := &restapi.EvictionRunResult{Counts: []restapi.EntityEvictionCount{}}
	for _, entityType := range order {
		result.Counts = append(result.Counts, restapi.EntityEvictionCount{Type: entityType, Count: int64(counts[entityType])})
	}
	s.Logger.Info("ran cleanup job", zap.String("group", string(req.GroupName)), zap.Bool("delete", run.Delete), zap.Any("counts", counts))
	return restapi.CreateEvictionRunResponseSet{Response200: result}
}

// evictor returns the evictor of the Raft group or a dsserr.NotFound error.
func (s *Server) evictor(name restapi.RaftGroupName) (Evictor, error) {
	evictor, ok := s.evictors[string(name)]
	if !ok {
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "No eviction for raft group %q", name)
	}
	return evictor, nil
}

// toEvictionConfig returns config with its entity types in the order of entityTypes.
func toEvictionConfig(entityTypes []string, config EvictionConfig) *restapi.EvictionConfig {
	result := &restapi.EvictionConfig{Interval: config.Interval.String(), Entities: []restapi.EntityEviction{}}
	for _, entityType := range entityTypes {
		if e, ok := config.Entities[entityType]; ok {
			result.Entities = append(result.Entities, restapi.EntityEviction{Type: entityType, Ttl: e.TTL.String(), Limit: new(int64(e.Limit))})
		}
	}
	return result
}

// fromEvictionConfig converts and validates config, or returns a dsserr.BadRequest error.
func fromEvictionConfig(entityTypes []string, config *restapi.EvictionConfig) (EvictionConfig, error) {
	interval, err := time.ParseDuration(config.Interval)
	if err != nil || interval <= 0 {
		return EvictionConfig{}, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid interval %q, expected a positive duration", config.Interval)
	}
	entities, _, err := fromEntityEvictions(entityTypes, config.Entities)
	if err != nil {
		return EvictionConfig{}, err
	}
	return EvictionConfig{Interval: interval, Entities: entities}, nil
}

// fromEvictionRunParameters converts and validates params, or returns a dsserr.BadRequest error. It also returns the
// entity types in the order of params.
func fromEvictionRunParameters(entityTypes []string, params *restapi.EvictionRunParameters) (EvictionRun, []string, error) {
	run := EvictionRun{Delete: params.Delete != nil && *params.Delete}
	entities, order, err := fromEntityEvictions(entityTypes, params.Entities)
	if err != nil {
		return EvictionRun{}, nil, err
	}
	if !run.Delete {
		for _, e := range params.Entities {
			if e.Limit != nil && *e.Limit != 0 {
				return EvictionRun{}, nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Limit set for %s while only counting, expected it only when deleting", e.Type)
			}
		}
	}
	run.Entities = entities
	return run, order, nil
}

// fromEntityEvictions converts and validates entities, or returns a dsserr.BadRequest error. It also returns the
// entity types in the order of entities.
func fromEntityEvictions(entityTypes []string, entities []restapi.EntityEviction) (map[string]EntityEviction, []string, error) {
	result := make(map[string]EntityEviction, len(entities))
	order := make([]string, 0, len(entities))
	for _, e := range entities {
		if !slices.Contains(entityTypes, e.Type) {
			return nil, nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Unknown entity type %q, expected one of %v", e.Type, entityTypes)
		}
		if _, ok := result[e.Type]; ok {
			return nil, nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Entity type %s listed more than once", e.Type)
		}
		ttl, err := time.ParseDuration(e.Ttl)
		if err != nil || ttl <= 0 {
			return nil, nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid TTL %q for %s, expected a positive duration", e.Ttl, e.Type)
		}
		var limit int64
		if e.Limit != nil {
			limit = *e.Limit
		}
		if limit < 0 {
			return nil, nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid limit %d for %s, expected 0 (unlimited) or more", limit, e.Type)
		}
		result[e.Type] = EntityEviction{TTL: ttl, Limit: int(limit)}
		order = append(order, e.Type)
	}
	return result, order, nil
}
