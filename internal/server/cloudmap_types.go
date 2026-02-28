package server

type cloudMapDataType struct {
	Name string
}

// AWS Cloud Map data types sourced from:
// https://docs.aws.amazon.com/cloud-map/latest/api/API_Types.html
var cloudMapDataTypes = []cloudMapDataType{
	{Name: "DnsConfig"},
	{Name: "DnsConfigChange"},
	{Name: "DnsProperties"},
	{Name: "DnsRecord"},
	{Name: "HealthCheckConfig"},
	{Name: "HealthCheckCustomConfig"},
	{Name: "HttpInstanceSummary"},
	{Name: "HttpNamespaceChange"},
	{Name: "HttpProperties"},
	{Name: "Instance"},
	{Name: "InstanceSummary"},
	{Name: "Namespace"},
	{Name: "NamespaceFilter"},
	{Name: "NamespaceProperties"},
	{Name: "NamespaceSummary"},
	{Name: "Operation"},
	{Name: "OperationFilter"},
	{Name: "OperationSummary"},
	{Name: "PrivateDnsNamespaceChange"},
	{Name: "PrivateDnsNamespaceProperties"},
	{Name: "PrivateDnsNamespacePropertiesChange"},
	{Name: "PrivateDnsPropertiesMutable"},
	{Name: "PrivateDnsPropertiesMutableChange"},
	{Name: "PublicDnsNamespaceChange"},
	{Name: "PublicDnsNamespaceProperties"},
	{Name: "PublicDnsNamespacePropertiesChange"},
	{Name: "PublicDnsPropertiesMutable"},
	{Name: "PublicDnsPropertiesMutableChange"},
	{Name: "Service"},
	{Name: "ServiceAttributes"},
	{Name: "ServiceChange"},
	{Name: "ServiceFilter"},
	{Name: "ServiceSummary"},
	{Name: "SOA"},
	{Name: "SOAChange"},
	{Name: "Tag"},
	{Name: "UpdateServiceAttributes"},
}

var cloudMapDataTypeByName = func() map[string]cloudMapDataType {
	out := make(map[string]cloudMapDataType, len(cloudMapDataTypes))
	for _, dt := range cloudMapDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
