package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../personalize-runtime/2018-05-22/service-2.json internal/awsmodels/personalizeruntime/model.json

//go:embed personalizeruntime/model.json
var personalizeruntimeModelJSON []byte

var personalizeruntimeModel = sync.OnceValue(func() *Model { return mustLoadModel("personalize-runtime", personalizeruntimeModelJSON) })

// PersonalizeRuntimeModel returns the validation model for Amazon PersonalizeRuntime.
func PersonalizeRuntimeModel() *Model { return personalizeruntimeModel() }
