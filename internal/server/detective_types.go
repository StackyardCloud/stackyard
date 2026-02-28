package server

type detectiveDataType struct {
	Name string
}

// Amazon Detective data types sourced from:
// https://docs.aws.amazon.com/detective/latest/APIReference/API_Types.html
var detectiveDataTypes = []detectiveDataType{
	{Name: "Account"},
	{Name: "Administrator"},
	{Name: "DatasourcePackageIngestDetail"},
	{Name: "DatasourcePackageUsageInfo"},
	{Name: "DateFilter"},
	{Name: "FilterCriteria"},
	{Name: "FlaggedIpAddressDetail"},
	{Name: "Graph"},
	{Name: "ImpossibleTravelDetail"},
	{Name: "Indicator"},
	{Name: "IndicatorDetail"},
	{Name: "InvestigationDetail"},
	{Name: "MemberDetail"},
	{Name: "MembershipDatasources"},
	{Name: "NewAsoDetail"},
	{Name: "NewGeolocationDetail"},
	{Name: "NewUserAgentDetail"},
	{Name: "RelatedFindingDetail"},
	{Name: "RelatedFindingGroupDetail"},
	{Name: "SortCriteria"},
	{Name: "StringFilter"},
	{Name: "TTPsObservedDetail"},
	{Name: "TimestampForCollection"},
	{Name: "UnprocessedAccount"},
	{Name: "UnprocessedGraph"},
	{Name: "UpdateOrganizationConfiguration"},
}

var detectiveDataTypeByName = func() map[string]detectiveDataType {
	out := make(map[string]detectiveDataType, len(detectiveDataTypes))
	for _, dt := range detectiveDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
