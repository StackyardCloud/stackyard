package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../machinelearning/2014-12-12/service-2.json internal/awsmodels/machinelearning/model.json

//go:embed machinelearning/model.json
var machineLearningModelJSON []byte

var machineLearningModel = sync.OnceValue(func() *Model { return mustLoadModel("machinelearning", machineLearningModelJSON) })

// MachineLearningModel returns the validation model for Amazon Machine Learning.
func MachineLearningModel() *Model { return machineLearningModel() }
