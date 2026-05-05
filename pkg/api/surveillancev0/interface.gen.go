// This file is auto-generated; do not change as any changes will be overwritten
package surveillancev0

import (
	"context"
	"github.com/interuss/dss/pkg/api"
)

var (
	SurveillanceDisplayProviderScope     = api.RequiredScope("surveillance.display_provider")
	SurveillanceServiceProviderScope     = api.RequiredScope("surveillance.service_provider")
	SearchTrafficSurveilledAreasSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceDisplayProviderScope},
		},
	}
	GetTrafficSurveilledAreaSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceDisplayProviderScope},
		},
		{
			"Authority": {SurveillanceServiceProviderScope},
		},
	}
	CreateTrafficSurveilledAreaSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceServiceProviderScope},
		},
	}
	UpdateTrafficSurveilledAreaSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceServiceProviderScope},
		},
	}
	DeleteTrafficSurveilledAreaSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceServiceProviderScope},
		},
	}
	SearchSubscriptionsSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceDisplayProviderScope},
		},
	}
	GetSubscriptionSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceDisplayProviderScope},
		},
		{
			"Authority": {SurveillanceServiceProviderScope},
		},
	}
	CreateSubscriptionSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceDisplayProviderScope},
		},
	}
	UpdateSubscriptionSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceDisplayProviderScope},
		},
	}
	DeleteSubscriptionSecurity = []api.AuthorizationOption{
		{
			"Authority": {SurveillanceDisplayProviderScope},
		},
	}
)

type SearchTrafficSurveilledAreasRequest struct {
	// The area in which to search for Traffic Surveilled Areas.  Some Traffic Surveilled Areas near this area but wholly outside it may also be returned.
	Area *GeoPolygonString

	// If specified, indicates non-interest in any Traffic Surveilled Areas that end before this time.  RFC 3339 format, per OpenAPI specification. The time zone must be 'Z'.
	EarliestTime *string

	// If specified, indicates non-interest in any Traffic Surveilled Areas that start after this time.  RFC 3339 format, per OpenAPI specification. The time zone must be 'Z'.
	LatestTime *string

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type SearchTrafficSurveilledAreasResponseSet struct {
	// Traffic Surveilled Areas were successfully retrieved.
	Response200 *SearchTrafficSurveilledAreasResponse

	// One or more input parameters were missing or invalid.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// The access token was decoded successfully but did not include a scope appropriate to this endpoint.
	Response403 *ErrorResponse

	// The requested area was too large.
	Response413 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type GetTrafficSurveilledAreaRequest struct {
	// EntityUUID of the Traffic Surveilled Area.
	Id EntityUUID

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type GetTrafficSurveilledAreaResponseSet struct {
	// Full information of the Traffic Surveilled Area was retrieved successfully.
	Response200 *GetTrafficSurveilledAreaResponse

	// One or more input parameters were missing or invalid.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// The access token was decoded successfully but did not include a scope appropriate to this endpoint.
	Response403 *ErrorResponse

	// The requested Entity could not be found.
	Response404 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type CreateTrafficSurveilledAreaRequest struct {
	// EntityUUID of the Traffic Surveilled Area.
	Id EntityUUID

	// The data contained in the body of this request, if it parsed correctly
	Body *CreateTrafficSurveilledAreaParameters

	// The error encountered when attempting to parse the body of this request
	BodyParseError error

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type CreateTrafficSurveilledAreaResponseSet struct {
	// An existing Traffic Surveilled Area was created successfully in the DSS.
	Response200 *PutTrafficSurveilledAreaResponse

	// * One or more input parameters were missing or invalid.
	// * The request attempted to mutate the Traffic Surveilled Area in a disallowed way.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// The access token was decoded successfully but did not include a scope appropriate to this endpoint.
	Response403 *ErrorResponse

	// * An Traffic Surveilled Area with the specified ID already exists and is owned by a different client.
	// * Despite repeated attempts, the DSS was unable to update the DAR because of other simultaneous changes.
	Response409 *ErrorResponse

	// The area of the Traffic Surveilled Area is too large.
	Response413 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type UpdateTrafficSurveilledAreaRequest struct {
	// EntityUUID of the Traffic Surveilled Area.
	Id EntityUUID

	// Version string used to reference an Traffic Surveilled Area at a particular point in time. Any updates to an existing Identification Service Area must contain the corresponding version to maintain idempotent updates.
	Version string

	// The data contained in the body of this request, if it parsed correctly
	Body *UpdateTrafficSurveilledAreaParameters

	// The error encountered when attempting to parse the body of this request
	BodyParseError error

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type UpdateTrafficSurveilledAreaResponseSet struct {
	// An existing Traffic Surveilled Area was updated successfully in the DSS.
	Response200 *PutTrafficSurveilledAreaResponse

	// * One or more input parameters were missing or invalid.
	// * The request attempted to mutate the Traffic Surveilled Area in a disallowed way.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// The access token was decoded successfully but did not include a scope appropriate to this endpoint.
	Response403 *ErrorResponse

	// * The specified Traffic Surveilled Area's current version does not match the provided version.
	// * Despite repeated attempts, the DSS was unable to update the DAR because of other simultaneous changes.
	Response409 *ErrorResponse

	// The area of the Traffic Surveilled Area is too large.
	Response413 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type DeleteTrafficSurveilledAreaRequest struct {
	// EntityUUID of the Traffic Surveilled Area.
	Id EntityUUID

	// Version string used to reference an Traffic Surveilled Area at a particular point in time. Any updates to an existing Identification Service Area must contain the corresponding version to maintain idempotent updates.
	Version string

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type DeleteTrafficSurveilledAreaResponseSet struct {
	// Traffic Surveilled Area was successfully deleted from DSS.
	Response200 *DeleteTrafficSurveilledAreaResponse

	// One or more input parameters were missing or invalid.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// * The access token was decoded successfully but did not include a scope appropriate to this endpoint.
	// * The Traffic Surveilled Area does not belong to the client requesting deletion.
	Response403 *ErrorResponse

	// Entity could not be deleted because it could not be found.
	Response404 *ErrorResponse

	// * The specified Traffic Surveilled Area's current version does not match the provided version.
	// * Despite repeated attempts, the DSS was unable to update the DAR because of other simultaneous changes.
	Response409 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type SearchSubscriptionsRequest struct {
	// The area in which to search for Subscriptions.  Some Subscriptions near this area but wholly outside it may also be returned.
	Area *GeoPolygonString

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type SearchSubscriptionsResponseSet struct {
	// Subscriptions were retrieved successfully.
	Response200 *SearchSubscriptionsResponse

	// One or more input parameters were missing or invalid.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// The access token was decoded successfully but did not include a scope appropriate to this endpoint.
	Response403 *ErrorResponse

	// The requested area was too large.
	Response413 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type GetSubscriptionRequest struct {
	// SubscriptionUUID of the subscription of interest.
	Id SubscriptionUUID

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type GetSubscriptionResponseSet struct {
	// Subscription information was retrieved successfully.
	Response200 *GetSubscriptionResponse

	// One or more input parameters were missing or invalid.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// The access token was decoded successfully but did not include a scope appropriate to this endpoint.
	Response403 *ErrorResponse

	// A Subscription with the specified ID was not found.
	Response404 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type CreateSubscriptionRequest struct {
	// SubscriptionUUID of the subscription of interest.
	Id SubscriptionUUID

	// The data contained in the body of this request, if it parsed correctly
	Body *CreateSubscriptionParameters

	// The error encountered when attempting to parse the body of this request
	BodyParseError error

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type CreateSubscriptionResponseSet struct {
	// A new Subscription was created successfully.
	Response200 *PutSubscriptionResponse

	// * One or more input parameters were missing or invalid.
	// * The request attempted to mutate the Subscription in a disallowed way.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// * The access token was decoded successfully but did not include a scope appropriate to this endpoint or the request.
	Response403 *ErrorResponse

	// * A Subscription with the specified ID already exists and is owned by a different client.
	// * Despite repeated attempts, the DSS was unable to update the DAR because of other simultaneous changes.
	Response409 *ErrorResponse

	// Client already has too many Subscriptions in the area where a new Subscription was requested.  To correct this problem, the client may query GET /subscriptions to see which Subscriptions are counting against their limit.  This problem should not generally be encountered because the Subscription limit should be above what any consumer that reasonably aggregates their Subscriptions should request.  But, a Subscription limit is necessary to bound performance requirements for DSS instances and would likely be hit by, e.g., a large surveillance display provider that created a Subscription for each of their display client users' views.
	Response429 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type UpdateSubscriptionRequest struct {
	// SubscriptionUUID of the subscription of interest.
	Id SubscriptionUUID

	// Version string used to reference a Subscription at a particular point in time. Any updates to an existing Subscription must contain the corresponding version to maintain idempotent updates.
	Version string

	// The data contained in the body of this request, if it parsed correctly
	Body *UpdateSubscriptionParameters

	// The error encountered when attempting to parse the body of this request
	BodyParseError error

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type UpdateSubscriptionResponseSet struct {
	// An existing Subscription was updated successfully.
	Response200 *PutSubscriptionResponse

	// * One or more input parameters were missing or invalid.
	// * The request attempted to mutate the Subscription in a disallowed way.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// * The access token was decoded successfully but did not include a scope appropriate to this endpoint or the request.
	Response403 *ErrorResponse

	// * The specified Subscriptions's current version does not match the provided version.
	// * Despite repeated attempts, the DSS was unable to update the DAR because of other simultaneous changes.
	Response409 *ErrorResponse

	// Client already has too many Subscriptions in the area where a new Subscription was requested.  To correct this problem, the client may query GET /subscriptions to see which Subscriptions are counting against their limit.  This problem should not generally be encountered because the Subscription limit should be above what any consumer that reasonably aggregates their Subscriptions should request.  But, a Subscription limit is necessary to bound performance requirements for DSS instances and would likely be hit by, e.g., a large surveillance display provider that created a Subscription for each of their display client users' views.
	Response429 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type DeleteSubscriptionRequest struct {
	// SubscriptionUUID of the subscription of interest.
	Id SubscriptionUUID

	// Version string used to reference a Subscription at a particular point in time. Any updates to an existing Subscription must contain the corresponding version to maintain idempotent updates.
	Version string

	// The result of attempting to authorize this request
	Auth api.AuthorizationResult
}
type DeleteSubscriptionResponseSet struct {
	// Subscription was deleted successfully.
	Response200 *DeleteSubscriptionResponse

	// One or more input parameters were missing or invalid.
	Response400 *ErrorResponse

	// Bearer access token was not provided in Authorization header, token could not be decoded, or token was invalid.
	Response401 *ErrorResponse

	// * The access token was decoded successfully but did not include a scope appropriate to this endpoint.
	// * The Entity does not belong to the client requesting deletion.
	Response403 *ErrorResponse

	// Subscription could not be deleted because it could not be found.
	Response404 *ErrorResponse

	// * The specified Subscriptions's current version does not match the provided version.
	// * Despite repeated attempts, the DSS was unable to update the DAR because of other simultaneous changes.
	Response409 *ErrorResponse

	// Auto-generated internal server error response
	Response500 *api.InternalServerErrorBody
}

type Implementation interface {
	// /dss/traffic_surveilled_areas
	// ---
	// Retrieve all Traffic Surveilled Areas in the DAR for a given area during the given time.  Note that some Traffic Surveilled Areas returned may lie entirely outside the requested area.
	SearchTrafficSurveilledAreas(ctx context.Context, req *SearchTrafficSurveilledAreasRequest) SearchTrafficSurveilledAreasResponseSet

	// /dss/traffic_surveilled_areas/{id}
	// ---
	// Retrieve full information of an Traffic Surveilled Area owned by the client.
	GetTrafficSurveilledArea(ctx context.Context, req *GetTrafficSurveilledAreaRequest) GetTrafficSurveilledAreaResponseSet

	// /dss/traffic_surveilled_areas/{id}
	// ---
	// Create a new Traffic Surveilled Area.  This call will fail if an Traffic Surveilled Area with the same ID already exists.
	CreateTrafficSurveilledArea(ctx context.Context, req *CreateTrafficSurveilledAreaRequest) CreateTrafficSurveilledAreaResponseSet

	// /dss/traffic_surveilled_areas/{id}/{version}
	// ---
	// Update an Traffic Surveilled Area.  The full content of the existing Traffic Surveilled Area will be replaced with the provided information as only the most recent version is retained.
	// Updating `time_start` is not allowed if it is before the current time.
	UpdateTrafficSurveilledArea(ctx context.Context, req *UpdateTrafficSurveilledAreaRequest) UpdateTrafficSurveilledAreaResponseSet

	// /dss/traffic_surveilled_areas/{id}/{version}
	// ---
	// Delete an Traffic Surveilled Area.
	DeleteTrafficSurveilledArea(ctx context.Context, req *DeleteTrafficSurveilledAreaRequest) DeleteTrafficSurveilledAreaResponseSet

	// /dss/subscriptions
	// ---
	// Retrieve subscriptions intersecting an area of interest.  Subscription notifications are only triggered by (and contain full information of) changes to, creation of, or deletion of, Entities referenced by or stored in the DSS; they do not involve any data transfer (such as surveillance flights updates) apart from Entity information.
	//
	// Only Subscriptions belonging to the caller are returned.  This endpoint would be used if a USS lost track of Subscriptions they had created and/or wanted to resolve an error indicating that they had too many existing Subscriptions in an area.
	SearchSubscriptions(ctx context.Context, req *SearchSubscriptionsRequest) SearchSubscriptionsResponseSet

	// /dss/subscriptions/{id}
	// ---
	// Verify the existence/validity and state of a particular subscription.
	GetSubscription(ctx context.Context, req *GetSubscriptionRequest) GetSubscriptionResponseSet

	// /dss/subscriptions/{id}
	// ---
	// Create a subscription.  This call will fail if a Subscription with the same ID already exists.
	//
	// Subscription notifications are only triggered by (and contain full information of) changes to, creation of, or deletion of, Entities referenced by or stored in the DSS; they do not involve any data transfer (such as flights updates) apart from Entity information.
	CreateSubscription(ctx context.Context, req *CreateSubscriptionRequest) CreateSubscriptionResponseSet

	// /dss/subscriptions/{id}/{version}
	// ---
	// Update a Subscription.  The full content of the existing Subscription will be replaced with the provided information as only the most recent version is retained.
	//
	// Subscription notifications are only triggered by (and contain full information of) changes to, creation of, or deletion of, Entities referenced by or stored in the DSS; they do not involve any data transfer (such as flights updates) apart from Entity information.
	UpdateSubscription(ctx context.Context, req *UpdateSubscriptionRequest) UpdateSubscriptionResponseSet

	// /dss/subscriptions/{id}/{version}
	// ---
	// Delete a subscription.
	DeleteSubscription(ctx context.Context, req *DeleteSubscriptionRequest) DeleteSubscriptionResponseSet
}
