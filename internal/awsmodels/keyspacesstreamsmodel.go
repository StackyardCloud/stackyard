package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../keyspacesstreams/2024-09-09/service-2.json internal/awsmodels/keyspacesstreams/model.json

//go:embed keyspacesstreams/model.json
var keyspacesstreamsModelJSON []byte

var keyspacesstreamsModel = sync.OnceValue(func() *Model { return mustLoadModel("keyspacesstreams", keyspacesstreamsModelJSON) })

// KeyspacesStreamsModel returns the validation model for Amazon Keyspaces Streams.
func KeyspacesStreamsModel() *Model { return keyspacesstreamsModel() }
