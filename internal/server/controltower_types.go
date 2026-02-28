package server

type controlTowerDataType struct {
	Name string
}

// AWS Control Tower data types sourced from:
// https://docs.aws.amazon.com/controltower/latest/APIReference/API_Types.html
var controlTowerDataTypes = []controlTowerDataType{
	{Name: "BaselineOperation"},
	{Name: "BaselineSummary"},
	{Name: "ControlOperation"},
	{Name: "ControlOperationFilter"},
	{Name: "ControlOperationSummary"},
	{Name: "DriftStatusSummary"},
	{Name: "EnabledBaselineDetails"},
	{Name: "EnabledBaselineDriftStatusSummary"},
	{Name: "EnabledBaselineDriftTypes"},
	{Name: "EnabledBaselineFilter"},
	{Name: "EnabledBaselineInheritanceDrift"},
	{Name: "EnabledBaselineParameter"},
	{Name: "EnabledBaselineParameterSummary"},
	{Name: "EnabledBaselineSummary"},
	{Name: "EnabledControlDetails"},
	{Name: "EnabledControlDriftTypes"},
	{Name: "EnabledControlFilter"},
	{Name: "EnabledControlInheritanceDrift"},
	{Name: "EnabledControlParameter"},
	{Name: "EnabledControlParameterSummary"},
	{Name: "EnabledControlResourceDrift"},
	{Name: "EnabledControlSummary"},
	{Name: "EnablementStatusSummary"},
	{Name: "LandingZoneDetail"},
	{Name: "LandingZoneDriftStatusSummary"},
	{Name: "LandingZoneOperationDetail"},
	{Name: "LandingZoneOperationFilter"},
	{Name: "LandingZoneOperationSummary"},
	{Name: "LandingZoneSummary"},
	{Name: "Region"},
	{Name: "UpdateLandingZone"},
}

var controlTowerDataTypeByName = func() map[string]controlTowerDataType {
	out := make(map[string]controlTowerDataType, len(controlTowerDataTypes))
	for _, dt := range controlTowerDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
