package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.41 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../pi/2018-02-27/service-2.json internal/awsmodels/pi/model.json

//go:embed pi/model.json
var piModelJSON []byte

var piModel = sync.OnceValue(func() *Model { return mustLoadModel("pi", piModelJSON) })

// PIModel returns the validation model for Amazon RDS Performance Insights.
func PIModel() *Model { return piModel() }
