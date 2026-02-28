package server

type rtbFabricDataType struct {
	Name string
}

// AWS RTB Fabric data types sourced from:
// https://docs.aws.amazon.com/rtb-fabric/latest/APIReference/API_Types.html
// (The legacy APIReference path redirects to /latest/api/.)
var rtbFabricDataTypes = []rtbFabricDataType{
	{Name: "Action"},
	{Name: "AutoScalingGroupsConfiguration"},
	{Name: "EksEndpointsConfiguration"},
	{Name: "Filter"},
	{Name: "FilterCriterion"},
	{Name: "HeaderTagAction"},
	{Name: "LinkApplicationLogConfiguration"},
	{Name: "LinkApplicationLogSampling"},
	{Name: "LinkAttributes"},
	{Name: "LinkLogSettings"},
	{Name: "ListLinksResponseStructure"},
	{Name: "ManagedEndpointConfiguration"},
	{Name: "ModuleConfiguration"},
	{Name: "ModuleParameters"},
	{Name: "NoBidAction"},
	{Name: "NoBidModuleParameters"},
	{Name: "OpenRtbAttributeModuleParameters"},
	{Name: "RateLimiterModuleParameters"},
	{Name: "ResponderErrorMaskingForHttpCode"},
	{Name: "TrustStoreConfiguration"},
	{Name: "UpdateResponderGateway"},
}

var rtbFabricDataTypeByName = func() map[string]rtbFabricDataType {
	out := make(map[string]rtbFabricDataType, len(rtbFabricDataTypes))
	for _, dt := range rtbFabricDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
