package server

type cloudWatchInvestigationsDataType struct {
	Name string
}

// Amazon CloudWatch Investigations data types sourced from:
// https://docs.aws.amazon.com/cloudwatchinvestigations/latest/APIReference/API_Types.html
var cloudWatchInvestigationsDataTypes = []cloudWatchInvestigationsDataType{
	{Name: "CrossAccountConfiguration"},
	{Name: "EncryptionConfiguration"},
	{Name: "ListInvestigationGroupsModel"},
}

var cloudWatchInvestigationsDataTypeByName = func() map[string]cloudWatchInvestigationsDataType {
	out := make(map[string]cloudWatchInvestigationsDataType, len(cloudWatchInvestigationsDataTypes))
	for _, dt := range cloudWatchInvestigationsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
