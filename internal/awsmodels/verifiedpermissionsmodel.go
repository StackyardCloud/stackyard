package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.41 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../verifiedpermissions/2021-12-01/service-2.json internal/awsmodels/verifiedpermissions/model.json

//go:embed verifiedpermissions/model.json
var verifiedPermissionsModelJSON []byte

var verifiedPermissionsModel = sync.OnceValue(func() *Model {
	return mustLoadModel("verifiedpermissions", verifiedPermissionsModelJSON)
})

// VerifiedPermissionsModel returns the validation model for Amazon Verified Permissions.
func VerifiedPermissionsModel() *Model { return verifiedPermissionsModel() }
