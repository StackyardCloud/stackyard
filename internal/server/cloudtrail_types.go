package server

type cloudTrailDataType struct {
	Name string
}

// AWS CloudTrail data types sourced from:
// https://docs.aws.amazon.com/awscloudtrail/latest/APIReference/API_Types.html
var cloudTrailDataTypes = []cloudTrailDataType{
	{Name: "AdvancedEventSelector"},
	{Name: "AdvancedFieldSelector"},
	{Name: "AggregationConfiguration"},
	{Name: "Channel"},
	{Name: "ContextKeySelector"},
	{Name: "DashboardDetail"},
	{Name: "DataResource"},
	{Name: "Destination"},
	{Name: "Event"},
	{Name: "EventDataStore"},
	{Name: "EventSelector"},
	{Name: "ImportFailureListItem"},
	{Name: "ImportsListItem"},
	{Name: "ImportSource"},
	{Name: "ImportStatistics"},
	{Name: "IngestionStatus"},
	{Name: "InsightSelector"},
	{Name: "LookupAttribute"},
	{Name: "PartitionKey"},
	{Name: "PublicKey"},
	{Name: "Query"},
	{Name: "QueryStatistics"},
	{Name: "QueryStatisticsForDescribeQuery"},
	{Name: "RefreshSchedule"},
	{Name: "RefreshScheduleFrequency"},
	{Name: "RequestWidget"},
	{Name: "Resource"},
	{Name: "ResourceTag"},
	{Name: "S3ImportSource"},
	{Name: "SearchSampleQueriesSearchResult"},
	{Name: "SourceConfig"},
	{Name: "Tag"},
	{Name: "Trail"},
	{Name: "TrailInfo"},
	{Name: "Widget"},
	{Name: "UpdateTrail"},
}

var cloudTrailDataTypeByName = func() map[string]cloudTrailDataType {
	out := make(map[string]cloudTrailDataType, len(cloudTrailDataTypes))
	for _, dt := range cloudTrailDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
