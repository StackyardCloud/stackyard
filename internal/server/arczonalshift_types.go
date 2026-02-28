package server

type arcZonalShiftDataType struct {
	Name string
}

// Amazon Route 53 Application Recovery Controller Zonal Shift data types sourced from:
// https://docs.aws.amazon.com/arc-zonal-shift/latest/APIReference/API_Types.html
// AWS currently serves this API reference under /latest/api/.
var arcZonalShiftDataTypes = []arcZonalShiftDataType{
	{Name: "AppliedWeights"},
	{Name: "AutoshiftInResource"},
	{Name: "PracticeRun"},
	{Name: "ShiftedAwayMonitor"},
	{Name: "ZonalAutoshiftConfiguration"},
	{Name: "ZonalShiftInResource"},
	{Name: "ZonalShiftSummary"},
}

var arcZonalShiftDataTypeByName = func() map[string]arcZonalShiftDataType {
	out := make(map[string]arcZonalShiftDataType, len(arcZonalShiftDataTypes))
	for _, dt := range arcZonalShiftDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
