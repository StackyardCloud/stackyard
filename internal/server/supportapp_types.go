package server

type supportAppDataType struct {
	Name string
}

// AWS Support App in Slack data types sourced from:
// https://docs.aws.amazon.com/supportapp/latest/APIReference/API_Types.html
var supportAppDataTypes = []supportAppDataType{
	{Name: "SlackChannelConfiguration"},
	{Name: "SlackWorkspaceConfiguration"},
}

var supportAppDataTypeByName = func() map[string]supportAppDataType {
	out := make(map[string]supportAppDataType, len(supportAppDataTypes))
	for _, dt := range supportAppDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
