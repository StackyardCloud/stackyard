package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py --ops=<the operations in sagemaker_resources.go> .../sagemaker/2017-07-24/service-2.json internal/awsmodels/sagemaker/model.json

//go:embed sagemaker/model.json
var sagemakerModelJSON []byte

var sagemakerModel = sync.OnceValue(func() *Model { return mustLoadModel("sagemaker", sagemakerModelJSON) })

// SageMakerResourceModel returns the validation model for the SageMaker control-plane operations backed by real state (feature groups, device fleets, devices, edge deployment plans and trial components).
func SageMakerResourceModel() *Model { return sagemakerModel() }
