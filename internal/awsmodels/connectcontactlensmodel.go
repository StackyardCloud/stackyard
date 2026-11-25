package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../connect-contact-lens/2020-08-21/service-2.json internal/awsmodels/connectcontactlens/model.json

//go:embed connectcontactlens/model.json
var connectContactLensModelJSON []byte

var connectContactLensModel = sync.OnceValue(func() *Model { return mustLoadModel("connectcontactlens", connectContactLensModelJSON) })

// ConnectContactLensModel returns the validation model for Amazon Connect Contact Lens.
func ConnectContactLensModel() *Model { return connectContactLensModel() }
