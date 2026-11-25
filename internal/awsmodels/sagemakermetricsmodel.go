package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../sagemaker-metrics/2022-09-30/service-2.json internal/awsmodels/sagemakermetrics/model.json

//go:embed sagemakermetrics/model.json
var sagemakermetricsModelJSON []byte

var sagemakermetricsModel = sync.OnceValue(func() *Model { return mustLoadModel("sagemaker-metrics", sagemakermetricsModelJSON) })

// SageMakerMetricsModel returns the validation model for Amazon SageMaker Metrics.
func SageMakerMetricsModel() *Model { return sagemakermetricsModel() }
