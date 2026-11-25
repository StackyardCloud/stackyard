package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../taxsettings/2018-05-10/service-2.json internal/awsmodels/taxsettings/model.json

//go:embed taxsettings/model.json
var taxsettingsModelJSON []byte

var taxsettingsModel = sync.OnceValue(func() *Model { return mustLoadModel("taxsettings", taxsettingsModelJSON) })

// TaxSettingsModel returns the validation model for Tax Settings.
func TaxSettingsModel() *Model { return taxsettingsModel() }
