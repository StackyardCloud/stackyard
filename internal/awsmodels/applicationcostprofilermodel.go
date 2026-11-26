package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../applicationcostprofiler/2020-09-10/service-2.json internal/awsmodels/applicationcostprofiler/model.json

//go:embed applicationcostprofiler/model.json
var applicationCostProfilerModelJSON []byte

var applicationCostProfilerModel = sync.OnceValue(func() *Model { return mustLoadModel("applicationcostprofiler", applicationCostProfilerModelJSON) })

// ApplicationCostProfilerModel returns the validation model for AWS Application Cost Profiler.
func ApplicationCostProfilerModel() *Model { return applicationCostProfilerModel() }
