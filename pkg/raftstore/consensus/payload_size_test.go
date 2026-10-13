package consensus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/golang/geo/s2"
	"github.com/google/uuid"
	"github.com/interuss/dss/pkg/api"
	restapi "github.com/interuss/dss/pkg/api/scdv1"
	"github.com/interuss/dss/pkg/geo"
	dssmodels "github.com/interuss/dss/pkg/models"
	ridops "github.com/interuss/dss/pkg/rid/operations"
	scdops "github.com/interuss/dss/pkg/scd/operations"
	dssstore "github.com/interuss/dss/pkg/store"
	"github.com/stretchr/testify/require"
)

const payloadSizeCompressionThreshold = 1024

type jsonEnvelopeProposal struct {
	ID          string    `json:"id"`
	NodeID      uint64    `json:"node_id"`
	Timestamp   time.Time `json:"timestamp"`
	RequestType string    `json:"request_type"`
	Value       []byte    `json:"value"`
	ReadOnly    bool      `json:"read_only"`
}

func consecutiveCells(n int) []s2.CellID {
	cells := make([]s2.CellID, 0, n)
	cell := s2.CellIDFromLatLng(s2.LatLngFromDegrees(46.5, 6.6)).Parent(geo.DefaultMaximumCellLevel)
	for range n {
		cells = append(cells, cell)
		cell = cell.Next()
	}
	return cells
}

// encodeWithCells encodes the request as JSON and replaces every "Cells" field with the given cells.
func encodeWithCells(t *testing.T, request dssstore.OperationRequest, cells []s2.CellID) []byte {
	encoded, err := json.Marshal(request)
	require.NoError(t, err)
	if cells == nil {
		return encoded
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var payload any
	require.NoError(t, decoder.Decode(&payload))

	var replace func(v any) int
	replace = func(v any) int {
		replaced := 0
		switch v := v.(type) {
		case map[string]any:
			for k, child := range v {
				if k == "Cells" {
					v[k] = cells
					replaced++
				} else {
					replaced += replace(child)
				}
			}
		case []any:
			for _, child := range v {
				replaced += replace(child)
			}
		}
		return replaced
	}
	require.Equal(t, 1, replace(payload), "expected exactly one Cells field in %s payload", request.OperationID())

	value, err := json.Marshal(payload)
	require.NoError(t, err)
	return value
}

func TestPayloadSizes(t *testing.T) {
	now := time.Date(2026, time.October, 8, 12, 0, 0, 0, time.UTC)
	start, end := now.Add(time.Minute), now.Add(time.Hour)
	baseURL := "https://uss1.example.com/api/v1"
	clientID := "uss1"
	auth := &api.AuthorizationResult{ClientID: &clientID, Scopes: []string{"utm.strategic_coordination"}}

	scdTime := func(tm time.Time) *restapi.Time {
		return &restapi.Time{Value: tm.Format(time.RFC3339Nano), Format: "RFC3339"}
	}
	scdAltitude := func(v float64) *restapi.Altitude {
		return &restapi.Altitude{Value: v, Reference: "W84", Units: "M"}
	}
	scdVolume := restapi.Volume4D{TimeStart: scdTime(start), TimeEnd: scdTime(end), Volume: restapi.Volume3D{
		AltitudeLower: scdAltitude(0), AltitudeUpper: scdAltitude(120),
		OutlineCircle: &restapi.Circle{
			Center: &restapi.LatLngPoint{Lat: 46.5, Lng: 6.6},
			Radius: &restapi.Radius{Value: 100, Units: "M"},
		},
	}}

	altitudeLo, altitudeHi := float32(0), float32(120)
	ridVolume := &dssmodels.Volume4D{StartTime: &start, EndTime: &end, SpatialVolume: &dssmodels.Volume3D{
		AltitudeLo: &altitudeLo, AltitudeHi: &altitudeHi,
		Footprint: &dssmodels.GeoCircle{Center: dssmodels.LatLngPoint{Lat: 46.5, Lng: 6.6}, RadiusMeter: 100},
	}}

	ridCreateISA, err := ridops.NewInsertISAPayload(dssmodels.ID(uuid.NewString()), "uss1", baseURL, "locality-1", ridVolume, false)
	require.NoError(t, err)

	ridCreateSubscription, err := ridops.NewInsertSubscriptionPayload(dssmodels.ID(uuid.NewString()), "uss1", baseURL, "locality-1",
		&dssmodels.CellsVolume4D{Cells: s2.CellUnion(consecutiveCells(1)), StartTime: &start, EndTime: &end}, false)
	require.NoError(t, err)

	notifyForOperationalIntents := true
	scdCreateSubscription, err := scdops.NewCreateSubscriptionPayload(restapi.SubscriptionID(uuid.NewString()), "uss1", &restapi.PutSubscriptionParameters{
		Extents:                     scdVolume,
		UssBaseUrl:                  restapi.SubscriptionUssBaseURL(baseURL),
		NotifyForOperationalIntents: &notifyForOperationalIntents,
	}, false)
	require.NoError(t, err)

	scdCreateConstraint, err := scdops.NewCreateConstraintPayload(restapi.EntityID(uuid.NewString()), "uss1", &restapi.PutConstraintReferenceParameters{
		Extents:    []restapi.Volume4D{scdVolume},
		UssBaseUrl: restapi.ConstraintUssBaseURL(baseURL),
	}, false, now)
	require.NoError(t, err)

	scdCreateOIR, err := scdops.NewCreateOIRPayload(restapi.EntityID(uuid.NewString()), &restapi.PutOperationalIntentReferenceParameters{
		Extents:         []restapi.Volume4D{scdVolume},
		State:           "Accepted",
		UssBaseUrl:      restapi.OperationalIntentUssBaseURL(baseURL),
		NewSubscription: &restapi.ImplicitSubscriptionParameters{UssBaseUrl: restapi.SubscriptionUssBaseURL(baseURL)},
	}, auth, false, now)
	require.NoError(t, err)

	scdDeleteOIR := &restapi.DeleteOperationalIntentReferenceRequest{
		Entityid: restapi.EntityID(uuid.NewString()),
		Ovn:      restapi.EntityOVN(uuid.NewString() + "_" + uuid.NewString()),
		Auth:     *auth,
	}

	type sizeCase struct {
		name    string
		request dssstore.OperationRequest
		cells   []s2.CellID
	}
	cases := []sizeCase{
		{"RID delete ISA", ridops.NewDeleteISAPayload(dssmodels.ID(uuid.NewString()), "uss1", nil), nil},
		{"SCD delete OIR", scdDeleteOIR, nil},
	}
	for _, op := range []struct {
		name    string
		request dssstore.OperationRequest
	}{
		{"RID create ISA", ridCreateISA},
		{"RID create subscription", ridCreateSubscription},
		{"SCD create subscription", scdCreateSubscription},
		{"SCD create constraint", scdCreateConstraint},
		{"SCD create OIR", scdCreateOIR},
	} {
		for _, n := range []int{1, 10, 100, 1000} {
			cases = append(cases, sizeCase{op.name, op.request, consecutiveCells(n)})
		}
	}

	rows := [][]string{{
		"Case", "Cells", "Payload (B)",
		// "JSON envelope (B)",
		"Protobuf envelope (B)", "Compressed (B)", "Saved",
	}}
	savedPercent := func(before, after int) string {
		return fmt.Sprintf("%.1f%%", 100*float64(before-after)/float64(before))
	}
	for _, c := range cases {
		value := encodeWithCells(t, c.request, c.cells)
		proposal := Proposal{ID: uuid.NewString(), NodeID: 1, Timestamp: now, RequestType: c.request.OperationID(), Value: value}

		// jsonEnvelope, err := json.Marshal(jsonEnvelopeProposal{
		// 	ID:          proposal.ID,
		// 	NodeID:      proposal.NodeID,
		// 	Timestamp:   proposal.Timestamp,
		// 	RequestType: proposal.RequestType,
		// 	Value:       proposal.Value,
		// 	ReadOnly:    proposal.ReadOnly,
		// })
		// require.NoError(t, err)

		protobufEnvelope, err := proposal.encode(0)
		require.NoError(t, err)

		compressedEnvelope, err := proposal.encode(payloadSizeCompressionThreshold)
		require.NoError(t, err)

		cells := "-"
		if c.cells != nil {
			cells = fmt.Sprint(len(c.cells))
		}
		rows = append(rows, []string{
			c.name,
			cells,
			fmt.Sprint(len(value)),
			// fmt.Sprint(len(jsonEnvelope)),
			fmt.Sprint(len(protobufEnvelope)),
			fmt.Sprint(len(compressedEnvelope)),
			savedPercent(len(protobufEnvelope), len(compressedEnvelope)),
		})
	}

	fmt.Print(markdownTable(rows))
}

// markdownTable renders rows as a Markdown table whose first row is the header. The first column is left-aligned
// and the others right-aligned. Columns are padded so that the table is also readable as plain text.
func markdownTable(rows [][]string) string {
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			widths[i] = max(widths[i], utf8.RuneCountInString(cell), 4)
		}
	}

	var sb strings.Builder
	writeRow := func(row []string) {
		sb.WriteString("|")
		for i, cell := range row {
			padding := strings.Repeat(" ", widths[i]-utf8.RuneCountInString(cell))
			if i == 0 {
				_, _ = fmt.Fprintf(&sb, " %s%s |", cell, padding)
			} else {
				_, _ = fmt.Fprintf(&sb, " %s%s |", padding, cell)
			}
		}
		sb.WriteString("\n")
	}

	writeRow(rows[0])
	sb.WriteString("|")
	for i, width := range widths {
		if i == 0 {
			_, _ = fmt.Fprintf(&sb, " %s |", strings.Repeat("-", width))
		} else {
			_, _ = fmt.Fprintf(&sb, " %s: |", strings.Repeat("-", width-1))
		}
	}
	sb.WriteString("\n")
	for _, row := range rows[1:] {
		writeRow(row)
	}
	return sb.String()
}
