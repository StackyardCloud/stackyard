package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../chime-sdk-messaging/2021-05-15/service-2.json internal/awsmodels/chimesdkmessaging/model.json

//go:embed chimesdkmessaging/model.json
var chimesdkmessagingModelJSON []byte

var chimesdkmessagingModel = sync.OnceValue(func() *Model { return mustLoadModel("chime-sdk-messaging", chimesdkmessagingModelJSON) })

// ChimeSDKMessagingModel returns the validation model for Amazon Chime SDK Messaging.
func ChimeSDKMessagingModel() *Model { return chimesdkmessagingModel() }
