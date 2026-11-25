package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../budgets/2016-10-20/service-2.json internal/awsmodels/budgets/model.json

//go:embed budgets/model.json
var budgetsModelJSON []byte

var budgetsModel = sync.OnceValue(func() *Model { return mustLoadModel("budgets", budgetsModelJSON) })

// BudgetsModel returns the validation model for AWS Budgets.
func BudgetsModel() *Model { return budgetsModel() }
