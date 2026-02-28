package server

type mediaTailorOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Elemental MediaTailor operations sourced from:
// https://docs.aws.amazon.com/mediatailor/latest/apireference/API_Operations.html
var mediaTailorOperations = []mediaTailorOperation{
	{Name: "ConfigureLogsForChannel", Method: "PUT", URI: "/configureLogs/channel"},
	{Name: "ConfigureLogsForPlaybackConfiguration", Method: "PUT", URI: "/configureLogs/playbackConfiguration"},
	{Name: "CreateChannel", Method: "POST", URI: "/channel/{ChannelName}"},
	{Name: "CreateLiveSource", Method: "POST", URI: "/sourceLocation/{SourceLocationName}/liveSource/{LiveSourceName}"},
	{Name: "CreatePrefetchSchedule", Method: "POST", URI: "/prefetchSchedule/{PlaybackConfigurationName}/{Name}"},
	{Name: "CreateProgram", Method: "POST", URI: "/channel/{ChannelName}/program/{ProgramName}"},
	{Name: "CreateSourceLocation", Method: "POST", URI: "/sourceLocation/{SourceLocationName}"},
	{Name: "CreateVodSource", Method: "POST", URI: "/sourceLocation/{SourceLocationName}/vodSource/{VodSourceName}"},
	{Name: "DeleteChannel", Method: "DELETE", URI: "/channel/{ChannelName}"},
	{Name: "DeleteChannelPolicy", Method: "DELETE", URI: "/channel/{ChannelName}/policy"},
	{Name: "DeleteLiveSource", Method: "DELETE", URI: "/sourceLocation/{SourceLocationName}/liveSource/{LiveSourceName}"},
	{Name: "DeletePlaybackConfiguration", Method: "DELETE", URI: "/playbackConfiguration/{Name}"},
	{Name: "DeletePrefetchSchedule", Method: "DELETE", URI: "/prefetchSchedule/{PlaybackConfigurationName}/{Name}"},
	{Name: "DeleteProgram", Method: "DELETE", URI: "/channel/{ChannelName}/program/{ProgramName}"},
	{Name: "DeleteSourceLocation", Method: "DELETE", URI: "/sourceLocation/{SourceLocationName}"},
	{Name: "DeleteVodSource", Method: "DELETE", URI: "/sourceLocation/{SourceLocationName}/vodSource/{VodSourceName}"},
	{Name: "DescribeChannel", Method: "GET", URI: "/channel/{ChannelName}"},
	{Name: "DescribeLiveSource", Method: "GET", URI: "/sourceLocation/{SourceLocationName}/liveSource/{LiveSourceName}"},
	{Name: "DescribeProgram", Method: "GET", URI: "/channel/{ChannelName}/program/{ProgramName}"},
	{Name: "DescribeSourceLocation", Method: "GET", URI: "/sourceLocation/{SourceLocationName}"},
	{Name: "DescribeVodSource", Method: "GET", URI: "/sourceLocation/{SourceLocationName}/vodSource/{VodSourceName}"},
	{Name: "GetChannelPolicy", Method: "GET", URI: "/channel/{ChannelName}/policy"},
	{Name: "GetChannelSchedule", Method: "GET", URI: "/channel/{ChannelName}/schedule"},
	{Name: "GetPlaybackConfiguration", Method: "GET", URI: "/playbackConfiguration/{Name}"},
	{Name: "GetPrefetchSchedule", Method: "GET", URI: "/prefetchSchedule/{PlaybackConfigurationName}/{Name}"},
	{Name: "ListAlerts", Method: "GET", URI: "/alerts"},
	{Name: "ListChannels", Method: "GET", URI: "/channels"},
	{Name: "ListLiveSources", Method: "GET", URI: "/sourceLocation/{SourceLocationName}/liveSources"},
	{Name: "ListPlaybackConfigurations", Method: "GET", URI: "/playbackConfigurations"},
	{Name: "ListPrefetchSchedules", Method: "POST", URI: "/prefetchSchedule/{PlaybackConfigurationName}"},
	{Name: "ListSourceLocations", Method: "GET", URI: "/sourceLocations"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "ListVodSources", Method: "GET", URI: "/sourceLocation/{SourceLocationName}/vodSources"},
	{Name: "PutChannelPolicy", Method: "PUT", URI: "/channel/{ChannelName}/policy"},
	{Name: "PutPlaybackConfiguration", Method: "PUT", URI: "/playbackConfiguration"},
	{Name: "StartChannel", Method: "PUT", URI: "/channel/{ChannelName}/start"},
	{Name: "StopChannel", Method: "PUT", URI: "/channel/{ChannelName}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateChannel", Method: "PUT", URI: "/channel/{ChannelName}"},
	{Name: "UpdateLiveSource", Method: "PUT", URI: "/sourceLocation/{SourceLocationName}/liveSource/{LiveSourceName}"},
	{Name: "UpdateProgram", Method: "PUT", URI: "/channel/{ChannelName}/program/{ProgramName}"},
	{Name: "UpdateSourceLocation", Method: "PUT", URI: "/sourceLocation/{SourceLocationName}"},
	{Name: "UpdateVodSource", Method: "PUT", URI: "/sourceLocation/{SourceLocationName}/vodSource/{VodSourceName}"},
}

var mediaTailorOperationByName = func() map[string]mediaTailorOperation {
	out := make(map[string]mediaTailorOperation, len(mediaTailorOperations))
	for _, op := range mediaTailorOperations {
		out[op.Name] = op
	}
	return out
}()
