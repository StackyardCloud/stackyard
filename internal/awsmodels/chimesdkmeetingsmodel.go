package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../chime-sdk-meetings/2021-07-15/service-2.json internal/awsmodels/chimesdkmeetings/model.json

//go:embed chimesdkmeetings/model.json
var chimesdkmeetingsModelJSON []byte

var chimesdkmeetingsModel = sync.OnceValue(func() *Model { return mustLoadModel("chime-sdk-meetings", chimesdkmeetingsModelJSON) })

// ChimeSDKMeetingsModel returns the validation model for Amazon Chime SDK Meetings.
func ChimeSDKMeetingsModel() *Model { return chimesdkmeetingsModel() }
