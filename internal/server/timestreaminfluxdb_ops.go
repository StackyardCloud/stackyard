package server

type timestreamInfluxDBOperation struct {
	Name string
}

// Timestream for InfluxDB operations sourced from:
// https://docs.aws.amazon.com/ts-influxdb/latest/ts-influxdb-api/API_Operations.html
var timestreamInfluxDBOperations = []timestreamInfluxDBOperation{
	{Name: "CreateDbCluster"},
	{Name: "CreateDbInstance"},
	{Name: "CreateDbParameterGroup"},
	{Name: "DeleteDbCluster"},
	{Name: "DeleteDbInstance"},
	{Name: "GetDbCluster"},
	{Name: "GetDbInstance"},
	{Name: "GetDbParameterGroup"},
	{Name: "ListDbClusters"},
	{Name: "ListDbInstances"},
	{Name: "ListDbInstancesForCluster"},
	{Name: "ListDbParameterGroups"},
	{Name: "ListTagsForResource"},
	{Name: "RebootDbCluster"},
	{Name: "RebootDbInstance"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateDbCluster"},
	{Name: "UpdateDbInstance"},
}

var timestreamInfluxDBOperationByName = func() map[string]timestreamInfluxDBOperation {
	out := make(map[string]timestreamInfluxDBOperation, len(timestreamInfluxDBOperations))
	for _, op := range timestreamInfluxDBOperations {
		out[op.Name] = op
	}
	return out
}()
