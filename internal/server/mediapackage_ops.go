package server

type mediaPackageOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Elemental MediaPackage V2 Live API operations sourced from:
// https://docs.aws.amazon.com/mediapackage/latest/APIReference/API_Operations.html
var mediaPackageOperations = []mediaPackageOperation{
	{Name: "CancelHarvestJob", Method: "PUT", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}/harvestJob/{HarvestJobName}"},
	{Name: "CreateChannel", Method: "POST", URI: "/channelGroup/{ChannelGroupName}/channel"},
	{Name: "CreateChannelGroup", Method: "POST", URI: "/channelGroup"},
	{Name: "CreateHarvestJob", Method: "POST", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}/harvestJob"},
	{Name: "CreateOriginEndpoint", Method: "POST", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint"},
	{Name: "DeleteChannel", Method: "DELETE", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/"},
	{Name: "DeleteChannelGroup", Method: "DELETE", URI: "/channelGroup/{ChannelGroupName}"},
	{Name: "DeleteChannelPolicy", Method: "DELETE", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/policy"},
	{Name: "DeleteOriginEndpoint", Method: "DELETE", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}"},
	{Name: "DeleteOriginEndpointPolicy", Method: "DELETE", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}/policy"},
	{Name: "GetChannel", Method: "GET", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/"},
	{Name: "GetChannelGroup", Method: "GET", URI: "/channelGroup/{ChannelGroupName}"},
	{Name: "GetChannelPolicy", Method: "GET", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/policy"},
	{Name: "GetHarvestJob", Method: "GET", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}/harvestJob/{HarvestJobName}"},
	{Name: "GetOriginEndpoint", Method: "GET", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}"},
	{Name: "GetOriginEndpointPolicy", Method: "GET", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}/policy"},
	{Name: "ListChannelGroups", Method: "GET", URI: "/channelGroup"},
	{Name: "ListChannels", Method: "GET", URI: "/channelGroup/{ChannelGroupName}/channel"},
	{Name: "ListHarvestJobs", Method: "GET", URI: "/channelGroup/{ChannelGroupName}/harvestJob"},
	{Name: "ListOriginEndpoints", Method: "GET", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "PutChannelPolicy", Method: "PUT", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/policy"},
	{Name: "PutOriginEndpointPolicy", Method: "POST", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}/policy"},
	{Name: "ResetChannelState", Method: "POST", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/reset"},
	{Name: "ResetOriginEndpointState", Method: "POST", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}/reset"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateChannel", Method: "PUT", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/"},
	{Name: "UpdateChannelGroup", Method: "PUT", URI: "/channelGroup/{ChannelGroupName}"},
	{Name: "UpdateOriginEndpoint", Method: "PUT", URI: "/channelGroup/{ChannelGroupName}/channel/{ChannelName}/originEndpoint/{OriginEndpointName}"},
}

var mediaPackageOperationByName = func() map[string]mediaPackageOperation {
	out := make(map[string]mediaPackageOperation, len(mediaPackageOperations))
	for _, op := range mediaPackageOperations {
		out[op.Name] = op
	}
	return out
}()
