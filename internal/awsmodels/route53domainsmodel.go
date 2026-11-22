package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.41 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../route53domains/2014-05-15/service-2.json internal/awsmodels/route53domains/model.json

//go:embed route53domains/model.json
var route53DomainsModelJSON []byte

var route53DomainsModel = sync.OnceValue(func() *Model { return mustLoadModel("route53domains", route53DomainsModelJSON) })

// Route53DomainsModel returns the validation model for Amazon Route 53 Domains.
func Route53DomainsModel() *Model { return route53DomainsModel() }
