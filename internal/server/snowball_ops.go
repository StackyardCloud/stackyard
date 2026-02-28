package server

type snowballOperation struct {
	Name string
}

// AWS Snowball Edge operations sourced from:
// https://docs.aws.amazon.com/snowball/latest/api-reference/API_Operations.html
var snowballOperations = []snowballOperation{
	{Name: "CancelCluster"},
	{Name: "CancelJob"},
	{Name: "CreateAddress"},
	{Name: "CreateCluster"},
	{Name: "CreateJob"},
	{Name: "CreateLongTermPricing"},
	{Name: "CreateReturnShippingLabel"},
	{Name: "DescribeAddress"},
	{Name: "DescribeAddresses"},
	{Name: "DescribeCluster"},
	{Name: "DescribeJob"},
	{Name: "DescribeReturnShippingLabel"},
	{Name: "GetJobManifest"},
	{Name: "GetJobUnlockCode"},
	{Name: "GetSnowballUsage"},
	{Name: "GetSoftwareUpdates"},
	{Name: "ListClusterJobs"},
	{Name: "ListClusters"},
	{Name: "ListCompatibleImages"},
	{Name: "ListJobs"},
	{Name: "ListLongTermPricing"},
	{Name: "ListPickupLocations"},
	{Name: "ListServiceVersions"},
	{Name: "UpdateCluster"},
	{Name: "UpdateJob"},
	{Name: "UpdateJobShipmentState"},
	{Name: "UpdateLongTermPricing"},
}

var snowballOperationByName = func() map[string]snowballOperation {
	out := make(map[string]snowballOperation, len(snowballOperations))
	for _, op := range snowballOperations {
		out[op.Name] = op
	}
	return out
}()
