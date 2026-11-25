package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../frauddetector/2019-11-15/service-2.json internal/awsmodels/frauddetector/model.json

//go:embed frauddetector/model.json
var fraudDetectorModelJSON []byte

var fraudDetectorModel = sync.OnceValue(func() *Model { return mustLoadModel("frauddetector", fraudDetectorModelJSON) })

// FraudDetectorModel returns the validation model for Amazon Fraud Detector.
func FraudDetectorModel() *Model { return fraudDetectorModel() }
