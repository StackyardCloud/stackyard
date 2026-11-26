package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../cloud9/2017-09-23/service-2.json internal/awsmodels/cloud9/model.json

//go:embed cloud9/model.json
var cloud9ModelJSON []byte

var cloud9Model = sync.OnceValue(func() *Model { return mustLoadModel("cloud9", cloud9ModelJSON) })

// Cloud9Model returns the validation model for AWS Cloud9.
func Cloud9Model() *Model { return cloud9Model() }
