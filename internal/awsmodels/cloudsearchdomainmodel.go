package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../cloudsearchdomain/2013-01-01/service-2.json internal/awsmodels/cloudsearchdomain/model.json

//go:embed cloudsearchdomain/model.json
var cloudSearchDomainModelJSON []byte

var cloudSearchDomainModel = sync.OnceValue(func() *Model { return mustLoadModel("cloudsearchdomain", cloudSearchDomainModelJSON) })

// CloudSearchDomainModel returns the validation model for Amazon CloudSearch Domain.
func CloudSearchDomainModel() *Model { return cloudSearchDomainModel() }
