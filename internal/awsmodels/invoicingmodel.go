package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../invoicing/2024-12-01/service-2.json internal/awsmodels/invoicing/model.json

//go:embed invoicing/model.json
var invoicingModelJSON []byte

var invoicingModel = sync.OnceValue(func() *Model { return mustLoadModel("invoicing", invoicingModelJSON) })

// InvoicingModel returns the validation model for AWS Invoicing.
func InvoicingModel() *Model { return invoicingModel() }
