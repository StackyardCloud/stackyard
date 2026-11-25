package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../pinpoint-email/2018-07-26/service-2.json internal/awsmodels/pinpointemail/model.json

//go:embed pinpointemail/model.json
var pinpointemailModelJSON []byte

var pinpointemailModel = sync.OnceValue(func() *Model { return mustLoadModel("pinpoint-email", pinpointemailModelJSON) })

// PinpointEmailModel returns the validation model for Amazon Pinpoint Email.
func PinpointEmailModel() *Model { return pinpointemailModel() }
