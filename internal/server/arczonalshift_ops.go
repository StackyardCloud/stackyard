package server

type arcZonalShiftOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Route 53 Application Recovery Controller Zonal Shift operations sourced from:
// https://docs.aws.amazon.com/arc-zonal-shift/latest/APIReference/API_Operations.html
// AWS currently serves this API reference under /latest/api/.
var arcZonalShiftOperations = []arcZonalShiftOperation{
	{Name: "CancelPracticeRun", Method: "POST", URI: "/"},
	{Name: "CancelZonalShift", Method: "POST", URI: "/"},
	{Name: "CreatePracticeRunConfiguration", Method: "POST", URI: "/"},
	{Name: "DeletePracticeRunConfiguration", Method: "POST", URI: "/"},
	{Name: "GetAutoshiftObserverNotificationStatus", Method: "POST", URI: "/"},
	{Name: "GetManagedResource", Method: "POST", URI: "/"},
	{Name: "ListAutoshifts", Method: "POST", URI: "/"},
	{Name: "ListManagedResources", Method: "POST", URI: "/"},
	{Name: "ListZonalShifts", Method: "POST", URI: "/"},
	{Name: "StartPracticeRun", Method: "POST", URI: "/"},
	{Name: "StartZonalShift", Method: "POST", URI: "/"},
	{Name: "UpdateAutoshiftObserverNotificationStatus", Method: "POST", URI: "/"},
	{Name: "UpdatePracticeRunConfiguration", Method: "POST", URI: "/"},
	{Name: "UpdateZonalAutoshiftConfiguration", Method: "POST", URI: "/"},
	{Name: "UpdateZonalShift", Method: "POST", URI: "/"},
}

var arcZonalShiftOperationByName = func() map[string]arcZonalShiftOperation {
	out := make(map[string]arcZonalShiftOperation, len(arcZonalShiftOperations))
	for _, op := range arcZonalShiftOperations {
		out[op.Name] = op
	}
	return out
}()
