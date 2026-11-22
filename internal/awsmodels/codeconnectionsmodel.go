package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copies of the AWS CLI 2.41 models, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../codeconnections/2023-12-01/service-2.json internal/awsmodels/codeconnections/model.json
//   gen-aws-model-slim.py .../codestar-connections/2019-12-01/service-2.json internal/awsmodels/codestarconnections/model.json

//go:embed codeconnections/model.json
var codeConnectionsModelJSON []byte

//go:embed codestarconnections/model.json
var codeStarConnectionsModelJSON []byte

var codeConnectionsModel = sync.OnceValue(func() *Model { return mustLoadModel("codeconnections", codeConnectionsModelJSON) })

var codeStarConnectionsModel = sync.OnceValue(func() *Model {
	return mustLoadModel("codestar-connections", codeStarConnectionsModelJSON)
})

// CodeConnectionsModel returns the validation model for AWS CodeConnections.
func CodeConnectionsModel() *Model { return codeConnectionsModel() }

// CodeStarConnectionsModel returns the validation model for AWS CodeStar Connections.
func CodeStarConnectionsModel() *Model { return codeStarConnectionsModel() }

func mustLoadModel(name string, raw []byte) *Model {
	m, err := LoadModel(raw)
	if err != nil {
		panic("awsmodels: embedded " + name + " model is invalid: " + err.Error())
	}
	return m
}
