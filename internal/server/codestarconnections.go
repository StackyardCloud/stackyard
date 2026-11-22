package server

import "github.com/stackyard/stackyard/internal/awsmodels"

// codeStarConnectionsAPI is AWS CodeStar Connections, the original name of
// CodeConnections. Requests go through handleCodeConnectionsJSONRouter and
// share its store; only the target prefix, signing name, ARN service prefix
// and model differ. Bad input gets 400 ValidationException from the model
// check, and responses use Content-Type application/x-amz-json-1.0.
var codeStarConnectionsAPI = &ccAPI{
	service: "codestar-connections",
	targets: []string{"com.amazonaws.codestar.connections.CodeStar_connections_20191201.", "CodeStar_connections_20191201."},
	model:   awsmodels.CodeStarConnectionsModel,
	knownOps: func() map[string]bool {
		out := map[string]bool{}
		for _, op := range codestarconnectionsOperations {
			out[op.Name] = true
		}
		return out
	}(),
}
