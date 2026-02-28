package server

type controlTowerOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Control Tower actions sourced from:
// https://docs.aws.amazon.com/controltower/latest/APIReference/API_Operations.html
var controlTowerOperations = []controlTowerOperation{
	{Name: "CreateLandingZone", Method: "POST", URI: "/create-landingzone"},
	{Name: "DeleteLandingZone", Method: "POST", URI: "/delete-landingzone"},
	{Name: "DisableBaseline", Method: "POST", URI: "/disable-baseline"},
	{Name: "DisableControl", Method: "POST", URI: "/disable-control"},
	{Name: "EnableBaseline", Method: "POST", URI: "/enable-baseline"},
	{Name: "EnableControl", Method: "POST", URI: "/enable-control"},
	{Name: "GetBaseline", Method: "POST", URI: "/get-baseline"},
	{Name: "GetBaselineOperation", Method: "POST", URI: "/get-baseline-operation"},
	{Name: "GetControlOperation", Method: "POST", URI: "/get-control-operation"},
	{Name: "GetEnabledBaseline", Method: "POST", URI: "/get-enabled-baseline"},
	{Name: "GetEnabledControl", Method: "POST", URI: "/get-enabled-control"},
	{Name: "GetLandingZone", Method: "POST", URI: "/get-landingzone"},
	{Name: "GetLandingZoneOperation", Method: "POST", URI: "/get-landingzone-operation"},
	{Name: "ListBaselines", Method: "POST", URI: "/list-baselines"},
	{Name: "ListControlOperations", Method: "POST", URI: "/list-control-operations"},
	{Name: "ListEnabledBaselines", Method: "POST", URI: "/list-enabled-baselines"},
	{Name: "ListEnabledControls", Method: "POST", URI: "/list-enabled-controls"},
	{Name: "ListLandingZoneOperations", Method: "POST", URI: "/list-landingzone-operations"},
	{Name: "ListLandingZones", Method: "POST", URI: "/list-landingzones"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ResetEnabledBaseline", Method: "POST", URI: "/reset-enabled-baseline"},
	{Name: "ResetEnabledControl", Method: "POST", URI: "/reset-enabled-control"},
	{Name: "ResetLandingZone", Method: "POST", URI: "/reset-landingzone"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateEnabledBaseline", Method: "POST", URI: "/update-enabled-baseline"},
	{Name: "UpdateEnabledControl", Method: "POST", URI: "/update-enabled-control"},
	{Name: "UpdateLandingZone", Method: "POST", URI: "/update-landingzone"},
}

var controlTowerOperationByName = func() map[string]controlTowerOperation {
	out := make(map[string]controlTowerOperation, len(controlTowerOperations))
	for _, op := range controlTowerOperations {
		out[op.Name] = op
	}
	return out
}()
