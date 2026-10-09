package models

import (
	"math/bits"
	"time"

	"github.com/golang/geo/s2"
	"github.com/google/uuid"
	"github.com/interuss/dss/pkg/models/modelspb"
	"github.com/interuss/stacktrace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func IDToProto(id ID) ([]byte, error) {
	parsed, err := uuid.Parse(id.String())
	if err != nil {
		return nil, stacktrace.Propagate(err, "invalid ID %q", id)
	}
	return parsed[:], nil
}

func IDFromProto(b []byte) (ID, error) {
	parsed, err := uuid.FromBytes(b)
	if err != nil {
		return "", stacktrace.Propagate(err, "invalid ID")
	}
	return ID(parsed.String()), nil
}

func VersionToProto(v *Version) *string {
	if v == nil {
		return nil
	}
	s := v.String()
	return &s
}

func VersionFromProto(s *string) (*Version, error) {
	if s == nil {
		return nil, nil
	}
	return VersionFromString(*s)
}

func CellsVolume4DToProto(v *CellsVolume4D) (*modelspb.CellsVolume4D, error) {
	if v == nil {
		return nil, nil
	}
	startTime, err := timeToProto(v.StartTime)
	if err != nil {
		return nil, stacktrace.Propagate(err, "invalid start time")
	}
	endTime, err := timeToProto(v.EndTime)
	if err != nil {
		return nil, stacktrace.Propagate(err, "invalid end time")
	}
	return &modelspb.CellsVolume4D{
		Cells:      CellUnionToProto(v.Cells),
		StartTime:  startTime,
		EndTime:    endTime,
		AltitudeLo: v.AltitudeLo,
		AltitudeHi: v.AltitudeHi,
	}, nil
}

func CellsVolume4DFromProto(pb *modelspb.CellsVolume4D) (*CellsVolume4D, error) {
	if pb == nil {
		return nil, nil
	}
	cells, err := CellUnionFromProto(pb.GetCells())
	if err != nil {
		return nil, stacktrace.Propagate(err, "invalid cells")
	}
	startTime, err := timeFromProto(pb.GetStartTime())
	if err != nil {
		return nil, stacktrace.Propagate(err, "invalid start time")
	}
	endTime, err := timeFromProto(pb.GetEndTime())
	if err != nil {
		return nil, stacktrace.Propagate(err, "invalid end time")
	}
	return &CellsVolume4D{
		Cells:      cells,
		StartTime:  startTime,
		EndTime:    endTime,
		AltitudeLo: pb.AltitudeLo,
		AltitudeHi: pb.AltitudeHi,
	}, nil
}

// CellUnionToProto delta-encodes cells, preserving their order. It makes no assumption on the cell levels.
//
// The first cell is stored as is. Each following cell is stored as its difference with the previous one.
//
// Cells of a given level all end with the same bit pattern (a single 1 bit followed by zeros, whose position
// depends on the level), so their differences share trailing zero bits. delta_shift is the number of trailing
// zero bits common to all non-zero deltas, and every delta is shifted right by it so that neighbouring cells
// encode as small varints. It is 0 when there are no non-zero deltas, and it drops when cells of different levels are mixed.
//
// CellUnionFromProto reverses it with cells[i] = cells[i-1] + (deltas[i-1] << delta_shift).
func CellUnionToProto(cells s2.CellUnion) *modelspb.CellUnion {
	if len(cells) == 0 {
		return nil
	}

	deltas := make([]int64, len(cells)-1)
	shift := 64
	for i := 1; i < len(cells); i++ {
		delta := uint64(cells[i]) - uint64(cells[i-1])
		deltas[i-1] = int64(delta)
		shift = min(shift, bits.TrailingZeros64(delta))
	}
	if shift == 64 {
		shift = 0
	}
	for i := range deltas {
		deltas[i] >>= shift
	}

	return &modelspb.CellUnion{
		FirstCell:  uint64(cells[0]),
		DeltaShift: uint32(shift),
		Deltas:     deltas,
	}
}

func CellUnionFromProto(pb *modelspb.CellUnion) (s2.CellUnion, error) {
	if pb == nil {
		return nil, nil
	}
	if pb.GetDeltaShift() >= 64 {
		return nil, stacktrace.NewError("invalid delta shift %d", pb.GetDeltaShift())
	}

	cells := make(s2.CellUnion, len(pb.GetDeltas())+1)
	cells[0] = s2.CellID(pb.GetFirstCell())
	for i, delta := range pb.GetDeltas() {
		cells[i+1] = cells[i] + s2.CellID(uint64(delta)<<pb.GetDeltaShift())
	}
	return cells, nil
}

func timeToProto(t *time.Time) (*timestamppb.Timestamp, error) {
	if t == nil {
		return nil, nil
	}
	ts := timestamppb.New(*t)
	if err := ts.CheckValid(); err != nil {
		return nil, stacktrace.Propagate(err, "time %s cannot be encoded", t)
	}
	return ts, nil
}

func timeFromProto(ts *timestamppb.Timestamp) (*time.Time, error) {
	if ts == nil {
		return nil, nil
	}
	if err := ts.CheckValid(); err != nil {
		return nil, stacktrace.Propagate(err, "invalid timestamp")
	}
	t := ts.AsTime()
	return &t, nil
}
