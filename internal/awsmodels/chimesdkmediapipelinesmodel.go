package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../chime-sdk-media-pipelines/2021-07-15/service-2.json internal/awsmodels/chimesdkmediapipelines/model.json

//go:embed chimesdkmediapipelines/model.json
var chimesdkmediapipelinesModelJSON []byte

var chimesdkmediapipelinesModel = sync.OnceValue(func() *Model { return mustLoadModel("chime-sdk-media-pipelines", chimesdkmediapipelinesModelJSON) })

// ChimeSDKMediaPipelinesModel returns the validation model for Amazon Chime SDK Media Pipelines.
func ChimeSDKMediaPipelinesModel() *Model { return chimesdkmediapipelinesModel() }
