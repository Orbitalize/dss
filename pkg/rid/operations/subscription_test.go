package operations

import (
	"context"
	"testing"

	"github.com/interuss/dss/pkg/api"
	ridv1 "github.com/interuss/dss/pkg/api/ridv1"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/dss/pkg/geo/testdata"
	"github.com/interuss/stacktrace"
	"github.com/stretchr/testify/require"
)

func TestExecuteSearchSubscriptionsFailsIfOwnerMissingFromContext(t *testing.T) {
	_, err := ExecuteSearchSubscriptions(context.Background(), nil, &ridv1.SearchSubscriptionsRequest{
		Area: (*ridv1.GeoPolygonString)(&testdata.Loop),
	})

	require.Error(t, err)
	require.Equal(t, dsserr.PermissionDenied, stacktrace.GetCode(err))
}

func TestExecuteSearchSubscriptionsFailsForInvalidArea(t *testing.T) {
	_, err := ExecuteSearchSubscriptions(context.Background(), nil, &ridv1.SearchSubscriptionsRequest{
		Area: (*ridv1.GeoPolygonString)(&testdata.LoopWithOddNumberOfCoordinates),
		Auth: api.AuthorizationResult{ClientID: &testdata.Owner},
	})

	require.Error(t, err)
	require.Equal(t, dsserr.BadRequest, stacktrace.GetCode(err))
}
