package server

type groundStationOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Ground Station operations sourced from:
// https://docs.aws.amazon.com/ground-station/latest/APIReference/API_Operations.html
var groundStationOperations = []groundStationOperation{
	{Name: "CancelContact", Method: "DELETE", URI: "/contact/{contactId}"},
	{Name: "CreateConfig", Method: "POST", URI: "/config"},
	{Name: "CreateDataflowEndpointGroup", Method: "POST", URI: "/dataflowEndpointGroup"},
	{Name: "CreateDataflowEndpointGroupV2", Method: "POST", URI: "/dataflowEndpointGroupV2"},
	{Name: "CreateEphemeris", Method: "POST", URI: "/ephemeris"},
	{Name: "CreateMissionProfile", Method: "POST", URI: "/missionprofile"},
	{Name: "DeleteConfig", Method: "DELETE", URI: "/config/{configType}/{configId}"},
	{Name: "DeleteDataflowEndpointGroup", Method: "DELETE", URI: "/dataflowEndpointGroup/{dataflowEndpointGroupId}"},
	{Name: "DeleteEphemeris", Method: "DELETE", URI: "/ephemeris/{ephemerisId}"},
	{Name: "DeleteMissionProfile", Method: "DELETE", URI: "/missionprofile/{missionProfileId}"},
	{Name: "DescribeContact", Method: "GET", URI: "/contact/{contactId}"},
	{Name: "DescribeEphemeris", Method: "GET", URI: "/ephemeris/{ephemerisId}"},
	{Name: "GetAgentConfiguration", Method: "GET", URI: "/agent/{agentId}/configuration"},
	{Name: "GetAgentTaskResponseUrl", Method: "GET", URI: "/agentResponseUrl/{agentId}"},
	{Name: "GetConfig", Method: "GET", URI: "/config/{configType}/{configId}"},
	{Name: "GetDataflowEndpointGroup", Method: "GET", URI: "/dataflowEndpointGroup/{dataflowEndpointGroupId}"},
	{Name: "GetMinuteUsage", Method: "POST", URI: "/minute-usage"},
	{Name: "GetMissionProfile", Method: "GET", URI: "/missionprofile/{missionProfileId}"},
	{Name: "GetSatellite", Method: "GET", URI: "/satellite/{satelliteId}"},
	{Name: "ListConfigs", Method: "GET", URI: "/config"},
	{Name: "ListContacts", Method: "POST", URI: "/contacts"},
	{Name: "ListDataflowEndpointGroups", Method: "GET", URI: "/dataflowEndpointGroup"},
	{Name: "ListEphemerides", Method: "POST", URI: "/ephemerides"},
	{Name: "ListGroundStations", Method: "GET", URI: "/groundstation"},
	{Name: "ListMissionProfiles", Method: "GET", URI: "/missionprofile"},
	{Name: "ListSatellites", Method: "GET", URI: "/satellite"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "RegisterAgent", Method: "POST", URI: "/agent"},
	{Name: "ReserveContact", Method: "POST", URI: "/contact"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateAgentStatus", Method: "PUT", URI: "/agent/{agentId}"},
	{Name: "UpdateConfig", Method: "PUT", URI: "/config/{configType}/{configId}"},
	{Name: "UpdateEphemeris", Method: "PUT", URI: "/ephemeris/{ephemerisId}"},
	{Name: "UpdateMissionProfile", Method: "PUT", URI: "/missionprofile/{missionProfileId}"},
}

var groundStationOperationByName = func() map[string]groundStationOperation {
	out := make(map[string]groundStationOperation, len(groundStationOperations))
	for _, op := range groundStationOperations {
		out[op.Name] = op
	}
	return out
}()
