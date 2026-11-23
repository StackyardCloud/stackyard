package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.41 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../textract/2018-06-27/service-2.json internal/awsmodels/textract/model.json

//go:embed textract/model.json
var textractModelJSON []byte

var textractModel = sync.OnceValue(func() *Model { return mustLoadModel("textract", textractModelJSON) })

// TextractModel returns the validation model for Amazon Textract.
func TextractModel() *Model { return textractModel() }
