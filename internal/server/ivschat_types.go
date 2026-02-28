package server

type ivsChatDataType struct {
	Name string
}

// Amazon IVS Chat data types sourced from:
// https://docs.aws.amazon.com/ivs/latest/ChatAPIReference/API_Types.html
var ivsChatDataTypes = []ivsChatDataType{
	{Name: "CloudWatchLogsDestinationConfiguration"},
	{Name: "DestinationConfiguration"},
	{Name: "FirehoseDestinationConfiguration"},
	{Name: "LoggingConfigurationSummary"},
	{Name: "MessageReviewHandler"},
	{Name: "RoomSummary"},
	{Name: "S3DestinationConfiguration"},
	{Name: "ValidationExceptionField"},
}

var ivsChatDataTypeByName = func() map[string]ivsChatDataType {
	out := make(map[string]ivsChatDataType, len(ivsChatDataTypes))
	for _, dt := range ivsChatDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
