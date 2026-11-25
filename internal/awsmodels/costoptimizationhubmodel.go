package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../cost-optimization-hub/2022-07-26/service-2.json internal/awsmodels/costoptimizationhub/model.json

//go:embed costoptimizationhub/model.json
var costOptimizationHubModelJSON []byte

var costOptimizationHubModel = sync.OnceValue(func() *Model { return mustLoadModel("cost-optimization-hub", costOptimizationHubModelJSON) })

// CostOptimizationHubModel returns the validation model for AWS Cost Optimization Hub.
func CostOptimizationHubModel() *Model { return costOptimizationHubModel() }
