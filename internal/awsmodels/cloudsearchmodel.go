package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../cloudsearch/2013-01-01/service-2.json internal/awsmodels/cloudsearch/model.json

//go:embed cloudsearch/model.json
var cloudSearchModelJSON []byte

var cloudSearchModel = sync.OnceValue(func() *Model { return mustLoadModel("cloudsearch", cloudSearchModelJSON) })

// CloudSearchModel returns the validation model for Amazon CloudSearch.
func CloudSearchModel() *Model { return cloudSearchModel() }
