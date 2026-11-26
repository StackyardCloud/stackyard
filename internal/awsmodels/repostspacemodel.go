package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../repostspace/2022-05-13/service-2.json internal/awsmodels/repostspace/model.json

//go:embed repostspace/model.json
var repostspaceModelJSON []byte

var repostspaceModel = sync.OnceValue(func() *Model { return mustLoadModel("repostspace", repostspaceModelJSON) })

// RepostSpaceModel returns the validation model for AWS re:Post Private.
func RepostSpaceModel() *Model { return repostspaceModel() }
