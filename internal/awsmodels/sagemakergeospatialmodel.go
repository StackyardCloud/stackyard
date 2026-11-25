package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../sagemaker-geospatial/2020-05-27/service-2.json internal/awsmodels/sagemakergeospatial/model.json

//go:embed sagemakergeospatial/model.json
var sageMakerGeospatialModelJSON []byte

var sageMakerGeospatialModel = sync.OnceValue(func() *Model { return mustLoadModel("sagemakergeospatial", sageMakerGeospatialModelJSON) })

// SageMakerGeospatialModel returns the validation model for Amazon SageMaker geospatial capabilities.
func SageMakerGeospatialModel() *Model { return sageMakerGeospatialModel() }
