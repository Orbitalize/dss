package server

import (
	"context"

	"github.com/interuss/dss/pkg/api"
	restapi "github.com/interuss/dss/pkg/api/surveillancev0"
)

// DeleteSubscription deletes an existing subscription.
func (s *Server) DeleteSubscription(ctx context.Context, req *restapi.DeleteSubscriptionRequest,
) restapi.DeleteSubscriptionResponseSet {
	return restapi.DeleteSubscriptionResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}

// SearchSubscriptions queries for existing subscriptions in the given bounds.
func (s *Server) SearchSubscriptions(ctx context.Context, req *restapi.SearchSubscriptionsRequest,
) restapi.SearchSubscriptionsResponseSet {
	return restapi.SearchSubscriptionsResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}

// GetSubscription gets a single subscription based on ID.
func (s *Server) GetSubscription(ctx context.Context, req *restapi.GetSubscriptionRequest,
) restapi.GetSubscriptionResponseSet {
	return restapi.GetSubscriptionResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}

// CreateSubscription creates a single subscription.
func (s *Server) CreateSubscription(ctx context.Context, req *restapi.CreateSubscriptionRequest,
) restapi.CreateSubscriptionResponseSet {
	return restapi.CreateSubscriptionResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}

// UpdateSubscription updates a single subscription.
func (s *Server) UpdateSubscription(ctx context.Context, req *restapi.UpdateSubscriptionRequest,
) restapi.UpdateSubscriptionResponseSet {
	return restapi.UpdateSubscriptionResponseSet{Response500: &api.InternalServerErrorBody{ErrorMessage: "not implemented"}}
}
