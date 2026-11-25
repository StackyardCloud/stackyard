package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../iotsecuretunneling/2018-10-05/service-2.json internal/awsmodels/iotsecuretunneling/model.json

//go:embed iotsecuretunneling/model.json
var iotsecuretunnelingModelJSON []byte

var iotsecuretunnelingModel = sync.OnceValue(func() *Model { return mustLoadModel("iotsecuretunneling", iotsecuretunnelingModelJSON) })

// IoTSecureTunnelingModel returns the validation model for AWS IoT Secure Tunneling.
func IoTSecureTunnelingModel() *Model { return iotsecuretunnelingModel() }
