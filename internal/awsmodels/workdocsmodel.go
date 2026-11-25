package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../workdocs/2016-05-01/service-2.json internal/awsmodels/workdocs/model.json

//go:embed workdocs/model.json
var workdocsModelJSON []byte

var workdocsModel = sync.OnceValue(func() *Model { return mustLoadModel("workdocs", workdocsModelJSON) })

// WorkDocsModel returns the validation model for Amazon WorkDocs.
func WorkDocsModel() *Model { return workdocsModel() }
