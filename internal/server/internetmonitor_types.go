package server

type internetMonitorDataType struct {
	Name string
}

// Amazon Internet Monitor data types sourced from:
// https://docs.aws.amazon.com/internet-monitor/latest/api/API_Types.html
var internetMonitorDataTypes = []internetMonitorDataType{
	{Name: "AvailabilityMeasurement"},
	{Name: "ClientLocation"},
	{Name: "FilterParameter"},
	{Name: "HealthEvent"},
	{Name: "HealthEventsConfig"},
	{Name: "ImpactedLocation"},
	{Name: "InternetEventSummary"},
	{Name: "InternetHealth"},
	{Name: "InternetMeasurementsLogDelivery"},
	{Name: "LocalHealthEventsConfig"},
	{Name: "Monitor"},
	{Name: "Network"},
	{Name: "NetworkImpairment"},
	{Name: "PerformanceMeasurement"},
	{Name: "QueryField"},
	{Name: "RoundTripTime"},
	{Name: "S3Config"},
	{Name: "UpdateMonitor"},
}

var internetMonitorDataTypeByName = func() map[string]internetMonitorDataType {
	out := make(map[string]internetMonitorDataType, len(internetMonitorDataTypes))
	for _, dt := range internetMonitorDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
