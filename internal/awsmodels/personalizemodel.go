package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../personalize/2018-05-22/service-2.json internal/awsmodels/personalize/model.json

//go:embed personalize/model.json
var personalizeModelJSON []byte

var personalizeModel = sync.OnceValue(func() *Model { return mustLoadModel("personalize", personalizeModelJSON) })

// PersonalizeModel returns the validation model for Amazon Personalize.
func PersonalizeModel() *Model { return personalizeModel() }
