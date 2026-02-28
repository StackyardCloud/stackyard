package server

type serviceQuotasOperation struct {
	Name string
}

// AWS Service Quotas operations sourced from:
// https://docs.aws.amazon.com/servicequotas/2019-06-24/apireference/API_Operations.html
var serviceQuotasOperations = []serviceQuotasOperation{
	{Name: "AssociateServiceQuotaTemplate"},
	{Name: "CreateSupportCase"},
	{Name: "DeleteServiceQuotaIncreaseRequestFromTemplate"},
	{Name: "DisassociateServiceQuotaTemplate"},
	{Name: "GetAWSDefaultServiceQuota"},
	{Name: "GetAssociationForServiceQuotaTemplate"},
	{Name: "GetAutoManagementConfiguration"},
	{Name: "GetQuotaUtilizationReport"},
	{Name: "GetRequestedServiceQuotaChange"},
	{Name: "GetServiceQuota"},
	{Name: "GetServiceQuotaIncreaseRequestFromTemplate"},
	{Name: "ListAWSDefaultServiceQuotas"},
	{Name: "ListRequestedServiceQuotaChangeHistory"},
	{Name: "ListRequestedServiceQuotaChangeHistoryByQuota"},
	{Name: "ListServiceQuotaIncreaseRequestsInTemplate"},
	{Name: "ListServiceQuotas"},
	{Name: "ListServices"},
	{Name: "ListTagsForResource"},
	{Name: "PutServiceQuotaIncreaseRequestIntoTemplate"},
	{Name: "RequestServiceQuotaIncrease"},
	{Name: "StartAutoManagement"},
	{Name: "StartQuotaUtilizationReport"},
	{Name: "StopAutoManagement"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateAutoManagement"},
}

var serviceQuotasOperationByName = func() map[string]serviceQuotasOperation {
	out := make(map[string]serviceQuotasOperation, len(serviceQuotasOperations))
	for _, op := range serviceQuotasOperations {
		out[op.Name] = op
	}
	return out
}()
