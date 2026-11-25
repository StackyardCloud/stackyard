package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../snow-device-management/2021-08-04/service-2.json internal/awsmodels/snowdevicemanagement/model.json

//go:embed snowdevicemanagement/model.json
var snowdevicemanagementModelJSON []byte

var snowdevicemanagementModel = sync.OnceValue(func() *Model { return mustLoadModel("snowdevicemanagement", snowdevicemanagementModelJSON) })

// SnowDeviceManagementModel returns the validation model for AWS Snow Device Management.
func SnowDeviceManagementModel() *Model { return snowdevicemanagementModel() }
