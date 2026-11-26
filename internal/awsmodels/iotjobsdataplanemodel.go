package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../iot-jobs-data/2017-09-29/service-2.json internal/awsmodels/iotjobsdataplane/model.json

//go:embed iotjobsdataplane/model.json
var iotjobsdataplaneModelJSON []byte

var iotjobsdataplaneModel = sync.OnceValue(func() *Model { return mustLoadModel("iotjobsdataplane", iotjobsdataplaneModelJSON) })

// IoTJobsDataPlaneModel returns the validation model for the AWS IoT Jobs Data Plane.
func IoTJobsDataPlaneModel() *Model { return iotjobsdataplaneModel() }
