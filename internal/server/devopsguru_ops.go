package server

type devOpsGuruOperation struct {
	Name string
}

// Amazon DevOps Guru operations sourced from:
// https://docs.aws.amazon.com/devops-guru/latest/APIReference/API_Operations.html
var devOpsGuruOperations = []devOpsGuruOperation{
	{Name: "AddNotificationChannel"},
	{Name: "DeleteInsight"},
	{Name: "DescribeAccountHealth"},
	{Name: "DescribeAccountOverview"},
	{Name: "DescribeAnomaly"},
	{Name: "DescribeEventSourcesConfig"},
	{Name: "DescribeFeedback"},
	{Name: "DescribeInsight"},
	{Name: "DescribeOrganizationHealth"},
	{Name: "DescribeOrganizationOverview"},
	{Name: "DescribeOrganizationResourceCollectionHealth"},
	{Name: "DescribeResourceCollectionHealth"},
	{Name: "DescribeServiceIntegration"},
	{Name: "GetCostEstimation"},
	{Name: "GetResourceCollection"},
	{Name: "ListAnomaliesForInsight"},
	{Name: "ListAnomalousLogGroups"},
	{Name: "ListEvents"},
	{Name: "ListInsights"},
	{Name: "ListMonitoredResources"},
	{Name: "ListNotificationChannels"},
	{Name: "ListOrganizationInsights"},
	{Name: "ListRecommendations"},
	{Name: "PutFeedback"},
	{Name: "RemoveNotificationChannel"},
	{Name: "SearchInsights"},
	{Name: "SearchOrganizationInsights"},
	{Name: "StartCostEstimation"},
	{Name: "UpdateEventSourcesConfig"},
	{Name: "UpdateResourceCollection"},
	{Name: "UpdateServiceIntegration"},
}

var devOpsGuruOperationByName = func() map[string]devOpsGuruOperation {
	out := make(map[string]devOpsGuruOperation, len(devOpsGuruOperations))
	for _, op := range devOpsGuruOperations {
		out[op.Name] = op
	}
	return out
}()
