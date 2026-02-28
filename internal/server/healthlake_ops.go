package server

type healthLakeOperation struct {
	Name string
}

// Amazon HealthLake operations sourced from:
// https://docs.aws.amazon.com/healthlake/latest/APIReference/API_Operations.html
var healthLakeOperations = []healthLakeOperation{
	{Name: "CreateFHIRDatastore"},
	{Name: "DeleteFHIRDatastore"},
	{Name: "DescribeFHIRDatastore"},
	{Name: "DescribeFHIRExportJob"},
	{Name: "DescribeFHIRImportJob"},
	{Name: "ListFHIRDatastores"},
	{Name: "ListFHIRExportJobs"},
	{Name: "ListFHIRImportJobs"},
	{Name: "ListTagsForResource"},
	{Name: "StartFHIRExportJob"},
	{Name: "StartFHIRImportJob"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
}

var healthLakeOperationByName = func() map[string]healthLakeOperation {
	out := make(map[string]healthLakeOperation, len(healthLakeOperations))
	for _, op := range healthLakeOperations {
		out[op.Name] = op
	}
	return out
}()
