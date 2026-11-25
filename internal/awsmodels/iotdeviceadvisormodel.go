package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../iotdeviceadvisor/2020-09-18/service-2.json internal/awsmodels/iotdeviceadvisor/model.json

//go:embed iotdeviceadvisor/model.json
var iotdeviceadvisorModelJSON []byte

var iotdeviceadvisorModel = sync.OnceValue(func() *Model { return mustLoadModel("iotdeviceadvisor", iotdeviceadvisorModelJSON) })

// IoTDeviceAdvisorModel returns the validation model for AWS IoT Core Device Advisor.
func IoTDeviceAdvisorModel() *Model { return iotdeviceadvisorModel() }
