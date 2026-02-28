package server

type cloudWatchApplicationInsightsDataType struct {
	Name string
}

// Amazon CloudWatch Application Insights data types sourced from:
// https://docs.aws.amazon.com/cloudwatch/latest/APIReference/API_Types.html
var cloudWatchApplicationInsightsDataTypes = []cloudWatchApplicationInsightsDataType{
	{Name: "AddWorkload"},
	{Name: "ApplicationComponent"},
	{Name: "ApplicationInfo"},
	{Name: "ConfigurationEvent"},
	{Name: "DiscoveryTypeResult"},
	{Name: "LogPattern"},
	{Name: "Observation"},
	{Name: "Problem"},
	{Name: "ProblemObservation"},
	{Name: "RecommendationItem"},
}

var cloudWatchApplicationInsightsDataTypeByName = func() map[string]cloudWatchApplicationInsightsDataType {
	out := make(map[string]cloudWatchApplicationInsightsDataType, len(cloudWatchApplicationInsightsDataTypes))
	for _, dt := range cloudWatchApplicationInsightsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
