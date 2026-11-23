package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.41 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../waf-regional/2016-11-28/service-2.json internal/awsmodels/wafregional/model.json

//go:embed wafregional/model.json
var wafRegionalModelJSON []byte

var wafRegionalModel = sync.OnceValue(func() *Model { return mustLoadModel("waf-regional", wafRegionalModelJSON) })

// WAFRegionalModel returns the validation model for AWS WAF Classic Regional.
func WAFRegionalModel() *Model { return wafRegionalModel() }
