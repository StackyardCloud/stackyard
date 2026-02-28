package server

type serviceQuotasDataType struct {
	Name string
}

// AWS Service Quotas data types sourced from:
// https://docs.aws.amazon.com/servicequotas/2019-06-24/apireference/API_Types.html
var serviceQuotasDataTypes = []serviceQuotasDataType{
	{Name: "ErrorReason"},
	{Name: "MetricInfo"},
	{Name: "QuotaContextInfo"},
	{Name: "QuotaInfo"},
	{Name: "QuotaPeriod"},
	{Name: "QuotaUtilizationInfo"},
	{Name: "RequestedServiceQuotaChange"},
	{Name: "ServiceInfo"},
	{Name: "ServiceQuota"},
	{Name: "ServiceQuotaIncreaseRequestInTemplate"},
	{Name: "Tag"},
}

var serviceQuotasDataTypeByName = func() map[string]serviceQuotasDataType {
	out := make(map[string]serviceQuotasDataType, len(serviceQuotasDataTypes))
	for _, dt := range serviceQuotasDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
