package server

type evsDataType struct {
	Name string
}

// Amazon Elastic VMware Service data types sourced from:
// https://docs.aws.amazon.com/evs/latest/APIReference/API_Types.html
var evsDataTypes = []evsDataType{
	{Name: "Check"},
	{Name: "ConnectivityInfo"},
	{Name: "EipAssociation"},
	{Name: "Environment"},
	{Name: "EnvironmentSummary"},
	{Name: "Host"},
	{Name: "HostInfoForCreate"},
	{Name: "InitialVlanInfo"},
	{Name: "InitialVlans"},
	{Name: "InstanceTypeEsxVersionsInfo"},
	{Name: "LicenseInfo"},
	{Name: "NetworkInterface"},
	{Name: "Secret"},
	{Name: "ServiceAccessSecurityGroups"},
	{Name: "ValidationExceptionField"},
	{Name: "VcfHostnames"},
	{Name: "VcfVersionInfo"},
	{Name: "Vlan"},
}

var evsDataTypeByName = func() map[string]evsDataType {
	out := make(map[string]evsDataType, len(evsDataTypes))
	for _, dt := range evsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
