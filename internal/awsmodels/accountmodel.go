package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../account/2021-02-01/service-2.json internal/awsmodels/account/model.json

//go:embed account/model.json
var accountModelJSON []byte

var accountModel = sync.OnceValue(func() *Model { return mustLoadModel("account", accountModelJSON) })

// AccountModel returns the validation model for AWS Account.
func AccountModel() *Model { return accountModel() }
