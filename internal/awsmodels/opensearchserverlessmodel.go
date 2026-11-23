package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.41 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../opensearchserverless/2021-11-01/service-2.json internal/awsmodels/opensearchserverless/model.json

//go:embed opensearchserverless/model.json
var openSearchServerlessModelJSON []byte

var openSearchServerlessModel = sync.OnceValue(func() *Model {
	return mustLoadModel("opensearchserverless", openSearchServerlessModelJSON)
})

// OpenSearchServerlessModel returns the validation model for OpenSearch Serverless.
func OpenSearchServerlessModel() *Model { return openSearchServerlessModel() }
