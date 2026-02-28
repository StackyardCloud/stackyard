package server

type ivsMultitrackOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon IVS Multitrack Video Integration operations sourced from:
// https://docs.aws.amazon.com/ivs/latest/BroadcastSWIntegAPIReference/actions.html
var ivsMultitrackOperations = []ivsMultitrackOperation{
	{Name: "FindIngest", Method: "GET", URI: "/api/v2/FindIngest"},
	{Name: "GetClientConfiguration", Method: "POST", URI: "/api/v3/GetClientConfiguration"},
}

var ivsMultitrackOperationByName = func() map[string]ivsMultitrackOperation {
	out := make(map[string]ivsMultitrackOperation, len(ivsMultitrackOperations))
	for _, op := range ivsMultitrackOperations {
		out[op.Name] = op
	}
	return out
}()
