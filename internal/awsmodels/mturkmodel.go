package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../mturk/2017-01-17/service-2.json internal/awsmodels/mturk/model.json

//go:embed mturk/model.json
var mturkModelJSON []byte

var mturkModel = sync.OnceValue(func() *Model { return mustLoadModel("mturk", mturkModelJSON) })

// MTurkModel returns the validation model for Amazon Mechanical Turk.
func MTurkModel() *Model { return mturkModel() }
