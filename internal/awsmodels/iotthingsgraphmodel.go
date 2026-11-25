package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../iotthingsgraph/2018-09-06/service-2.json internal/awsmodels/iotthingsgraph/model.json

//go:embed iotthingsgraph/model.json
var iotthingsgraphModelJSON []byte

var iotthingsgraphModel = sync.OnceValue(func() *Model { return mustLoadModel("iotthingsgraph", iotthingsgraphModelJSON) })

// IoTThingsGraphModel returns the validation model for AWS IoT Things Graph.
func IoTThingsGraphModel() *Model { return iotthingsgraphModel() }
