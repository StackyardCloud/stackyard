package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../sagemaker-featurestore-runtime/2020-07-01/service-2.json internal/awsmodels/sagemakerfeaturestoreruntime/model.json

//go:embed sagemakerfeaturestoreruntime/model.json
var sagemakerfeaturestoreruntimeModelJSON []byte

var sagemakerfeaturestoreruntimeModel = sync.OnceValue(func() *Model {
	return mustLoadModel("sagemaker-featurestore-runtime", sagemakerfeaturestoreruntimeModelJSON)
})

// SageMakerFeatureStoreRuntimeModel returns the validation model for Amazon SageMaker Feature Store Runtime.
func SageMakerFeatureStoreRuntimeModel() *Model { return sagemakerfeaturestoreruntimeModel() }
