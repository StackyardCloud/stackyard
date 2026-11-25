package server

type iotSecureTunnelingOperation struct{ Name string }

// AWS IoT Secure Tunneling operations sourced from:
// https://docs.aws.amazon.com/iot/latest/apireference/API_Operations_AWS_IoT_Secure_Tunneling.html
var iotSecureTunnelingOperations = []iotSecureTunnelingOperation{
	{Name: "CloseTunnel"},
	{Name: "DescribeTunnel"},
	{Name: "ListTagsForResource"},
	{Name: "ListTunnels"},
	{Name: "OpenTunnel"},
	{Name: "RotateTunnelAccessToken"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
}

var iotSecureTunnelingOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range iotSecureTunnelingOperations {
		out[op.Name] = true
	}
	return out
}()
