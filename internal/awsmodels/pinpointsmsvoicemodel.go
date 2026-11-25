package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../pinpoint-sms-voice/2018-09-05/service-2.json internal/awsmodels/pinpointsmsvoice/model.json

//go:embed pinpointsmsvoice/model.json
var pinpointsmsvoiceModelJSON []byte

var pinpointsmsvoiceModel = sync.OnceValue(func() *Model { return mustLoadModel("pinpoint-sms-voice", pinpointsmsvoiceModelJSON) })

// PinpointSMSVoiceModel returns the validation model for Amazon Pinpoint SMS and Voice (v1).
func PinpointSMSVoiceModel() *Model { return pinpointsmsvoiceModel() }
