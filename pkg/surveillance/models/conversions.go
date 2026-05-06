package models

import (
	"time"

	ridv2restapi "github.com/interuss/dss/pkg/api/ridv2"
	ridv2api "github.com/interuss/dss/pkg/rid/models/api/v2"

	restapi "github.com/interuss/dss/pkg/api/surveillancev0"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/stacktrace"
)

// === Surveillance -> Business ===

// FromTime converts Surveillance v1 REST model to standard golang Time
func FromTime(t *restapi.Time) (*time.Time, error) {
	ridv2time := (*ridv2restapi.Time)(t)
	return ridv2api.FromTime(ridv2time)
}

// FromAltitude converts Surveillance v1 REST model to float
func FromAltitude(alt *restapi.Altitude) (*float32, error) {
	ridv2alt := (*ridv2restapi.Altitude)(alt)
	return ridv2api.FromAltitude(ridv2alt)
}

// FromVolume4D converts Surveillance v1 REST model to business object
func FromVolume4D(vol4 *restapi.Volume4D) (*dssmodels.Volume4D, error) {
	vol3, err := FromVolume3D(&vol4.Volume)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error parsing spatial volume of Volume4D")
	}

	result := &dssmodels.Volume4D{
		SpatialVolume: vol3,
	}

	result.StartTime, err = FromTime(vol4.TimeStart)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error parsing start time of Volume4D")
	}
	result.EndTime, err = FromTime(vol4.TimeEnd)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error parsing end time of Volume4D")
	}

	return result, nil
}

// FromVolume3D converts Surveillance v1 REST model to business object
func FromVolume3D(vol3 *restapi.Volume3D) (*dssmodels.Volume3D, error) {
	altitudeLo, err := FromAltitude(vol3.AltitudeLower)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error parsing lower altitude of Volume3D")
	}
	altitudeHi, err := FromAltitude(vol3.AltitudeUpper)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error parsing upper altitude of Volume3D")
	}

	if vol3.OutlinePolygon != nil {
		if vol3.OutlineCircle != nil {
			return nil, stacktrace.NewError("Only one of outline_circle or outline_polygon may be specified")
		}
		footprint := FromPolygon(vol3.OutlinePolygon)

		result := &dssmodels.Volume3D{
			Footprint:  footprint,
			AltitudeLo: altitudeLo,
			AltitudeHi: altitudeHi,
		}

		return result, nil
	}

	if vol3.OutlineCircle != nil {
		footprint, err := FromCircle(vol3.OutlineCircle)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Error parsing outline_circle for Volume3D")
		}

		result := &dssmodels.Volume3D{
			Footprint:  footprint,
			AltitudeLo: altitudeLo,
			AltitudeHi: altitudeHi,
		}

		return result, nil
	}

	return nil, stacktrace.NewError("Neither outline_polygon nor outline_circle were specified in volume")
}

// FromPolygon converts Surveillance v1 REST model to business object
func FromPolygon(polygon *restapi.Polygon) *dssmodels.GeoPolygon {
	result := &dssmodels.GeoPolygon{}

	for _, ltlng := range polygon.Vertices {
		result.Vertices = append(result.Vertices, FromLatLngPoint(&ltlng))
	}

	return result
}

// FromCircle converts Surveillance v1 REST model to business object
func FromCircle(circle *restapi.Circle) (*dssmodels.GeoCircle, error) {
	if circle.Center == nil {
		return nil, stacktrace.NewError("Missing `center` from circle")
	}
	if circle.Radius == nil {
		return nil, stacktrace.NewError("Missing `radius` from circle")
	}
	if circle.Radius.Units != "M" {
		return nil, stacktrace.NewError("Only circle radius units of 'M' are acceptable for UTM")
	}
	result := &dssmodels.GeoCircle{
		Center:      *FromLatLngPoint(circle.Center),
		RadiusMeter: circle.Radius.Value,
	}
	return result, nil
}

// FromLatLngPoint converts Surveillance v1 REST model to business object
func FromLatLngPoint(pt *restapi.LatLngPoint) *dssmodels.LatLngPoint {
	return &dssmodels.LatLngPoint{
		Lat: float64(pt.Lat),
		Lng: float64(pt.Lng),
	}
}

// === Business -> Surveillance ===

// ToTime converts standard golang Time to Surveillance v1 REST model
func ToTime(t *time.Time) *restapi.Time {
	return (*restapi.Time)(ridv2api.ToTime(t))
}

// ToTrafficSurveilledArea converts an TrafficSurveilledArea
// business object to Surveillance v1 REST model for API consumption.
func ToTrafficSurveilledArea(i *TrafficSurveilledArea) *restapi.TrafficSurveilledArea {
	result := &restapi.TrafficSurveilledArea{
		Id:         restapi.EntityUUID(i.ID.String()),
		Owner:      i.Owner.String(),
		UssBaseUrl: restapi.FlightsUSSBaseURL(i.URL),
		Version:    restapi.Version(i.Version.String()),
	}
	if i.StartTime != nil {
		result.TimeStart = *ToTime(i.StartTime)
	}
	if i.EndTime != nil {
		result.TimeEnd = *ToTime(i.EndTime)
	}

	return result
}

// MakeSubscribersToNotify groups the passed subscriptions by their callback URL,
// returning a collection of subscribers to notify that contains one entry per distinct callback URL.
func MakeSubscribersToNotify(subscriptions []*Subscription) []restapi.SubscriberToNotify {
	subscriptionsByURL := map[string][]restapi.SubscriptionState{}
	for _, sub := range subscriptions {
		notifIdx := restapi.SubscriptionNotificationIndex(sub.NotificationIndex)
		subState := restapi.SubscriptionState{
			SubscriptionId:    restapi.SubscriptionUUID(sub.ID),
			NotificationIndex: &notifIdx,
		}
		subscriptionsByURL[sub.URL] = append(subscriptionsByURL[sub.URL], subState)
	}

	result := []restapi.SubscriberToNotify{}
	for url, states := range subscriptionsByURL {
		result = append(result, restapi.SubscriberToNotify{
			Url:           restapi.URL(url),
			Subscriptions: states,
		})
	}

	return result
}

// ToSubscription converts a subscription business object to a Subscription
// Surveillance v1 REST model for API consumption.
func ToSubscription(s *Subscription) *restapi.Subscription {
	notifIdx := restapi.SubscriptionNotificationIndex(s.NotificationIndex)
	result := &restapi.Subscription{
		Id:                restapi.SubscriptionUUID(s.ID.String()),
		Owner:             s.Owner.String(),
		UssBaseUrl:        restapi.SubscriptionUSSBaseURL(s.URL),
		NotificationIndex: &notifIdx,
		Version:           restapi.Version(s.Version.String()),
		TimeStart:         ToTime(s.StartTime),
		TimeEnd:           ToTime(s.EndTime),
	}

	return result
}
