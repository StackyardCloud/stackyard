package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../bcm-pricing-calculator/2024-06-19/service-2.json internal/awsmodels/bcmpricingcalculator/model.json

//go:embed bcmpricingcalculator/model.json
var bcmPricingCalculatorModelJSON []byte

var bcmPricingCalculatorModel = sync.OnceValue(func() *Model { return mustLoadModel("bcmpricingcalculator", bcmPricingCalculatorModelJSON) })

// BCMPricingCalculatorModel returns the validation model for AWS Billing and Cost Management Pricing Calculator.
func BCMPricingCalculatorModel() *Model { return bcmPricingCalculatorModel() }
