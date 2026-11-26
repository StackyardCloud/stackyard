package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../outposts/2019-12-03/service-2.json internal/awsmodels/outposts/model.json

//go:embed outposts/model.json
var outpostsModelJSON []byte

var outpostsModel = sync.OnceValue(func() *Model { return mustLoadModel("outposts", outpostsModelJSON) })

// OutpostsModel returns the validation model for AWS Outposts.
func OutpostsModel() *Model { return outpostsModel() }
