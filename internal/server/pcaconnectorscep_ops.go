package server

type pcaconnectorscepOperation struct {
	Name   string
	Method string
	URI    string
}

// Private CA Connector for SCEP operations sourced from:
// https://docs.aws.amazon.com/C4SCEP_API/pca-connector-scep/latest/APIReference/API_Operations.html
var pcaconnectorscepOperations = []pcaconnectorscepOperation{
	{Name: "CreateChallenge", Method: "POST", URI: "/challenges"},
	{Name: "CreateConnector", Method: "POST", URI: "/connectors"},
	{Name: "DeleteChallenge", Method: "DELETE", URI: "/challenges/{ChallengeArn}"},
	{Name: "DeleteConnector", Method: "DELETE", URI: "/connectors/{ConnectorArn}"},
	{Name: "GetChallengeMetadata", Method: "GET", URI: "/challengeMetadata/{ChallengeArn}"},
	{Name: "GetChallengePassword", Method: "GET", URI: "/challengePasswords/{ChallengeArn}"},
	{Name: "GetConnector", Method: "GET", URI: "/connectors/{ConnectorArn}"},
	{Name: "ListChallengeMetadata", Method: "GET", URI: "/challengeMetadata"},
	{Name: "ListConnectors", Method: "GET", URI: "/connectors"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
}

var pcaconnectorscepOperationByName = func() map[string]pcaconnectorscepOperation {
	out := make(map[string]pcaconnectorscepOperation, len(pcaconnectorscepOperations))
	for _, op := range pcaconnectorscepOperations {
		out[op.Name] = op
	}
	return out
}()
