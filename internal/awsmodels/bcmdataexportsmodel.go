package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../bcm-data-exports/2023-11-26/service-2.json internal/awsmodels/bcmdataexports/model.json

//go:embed bcmdataexports/model.json
var bcmDataExportsModelJSON []byte

var bcmDataExportsModel = sync.OnceValue(func() *Model { return mustLoadModel("bcm-data-exports", bcmDataExportsModelJSON) })

// BCMDataExportsModel returns the validation model for AWS Billing and Cost Management Data Exports.
func BCMDataExportsModel() *Model { return bcmDataExportsModel() }
