package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../connectcases/2022-10-03/service-2.json internal/awsmodels/connectcases/model.json

//go:embed connectcases/model.json
var connectCasesModelJSON []byte

var connectCasesModel = sync.OnceValue(func() *Model { return mustLoadModel("connectcases", connectCasesModelJSON) })

// ConnectCasesModel returns the validation model for Amazon Connect Cases.
func ConnectCasesModel() *Model { return connectCasesModel() }
