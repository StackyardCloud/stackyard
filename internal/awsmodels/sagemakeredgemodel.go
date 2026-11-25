package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../sagemaker-edge/2020-09-23/service-2.json internal/awsmodels/sagemakeredge/model.json

//go:embed sagemakeredge/model.json
var sagemakeredgeModelJSON []byte

var sagemakeredgeModel = sync.OnceValue(func() *Model { return mustLoadModel("sagemaker-edge", sagemakeredgeModelJSON) })

// SageMakerEdgeModel returns the validation model for Amazon SageMaker Edge Manager.
func SageMakerEdgeModel() *Model { return sagemakeredgeModel() }
