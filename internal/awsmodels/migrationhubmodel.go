package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.41 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../mgh/2017-05-31/service-2.json internal/awsmodels/migrationhub/model.json

//go:embed migrationhub/model.json
var migrationHubModelJSON []byte

var migrationHubModel = sync.OnceValue(func() *Model { return mustLoadModel("mgh", migrationHubModelJSON) })

// MigrationHubModel returns the validation model for AWS Migration Hub.
func MigrationHubModel() *Model { return migrationHubModel() }
