package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../freetier/2023-09-07/service-2.json internal/awsmodels/freetier/model.json

//go:embed freetier/model.json
var freetierModelJSON []byte

var freetierModel = sync.OnceValue(func() *Model { return mustLoadModel("freetier", freetierModelJSON) })

// FreeTierModel returns the validation model for AWS Free Tier.
func FreeTierModel() *Model { return freetierModel() }
