package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../lookoutequipment/2020-12-15/service-2.json internal/awsmodels/lookoutequipment/model.json

//go:embed lookoutequipment/model.json
var lookoutEquipmentModelJSON []byte

var lookoutEquipmentModel = sync.OnceValue(func() *Model { return mustLoadModel("lookoutequipment", lookoutEquipmentModelJSON) })

// LookoutEquipmentModel returns the validation model for Amazon Lookout for Equipment.
func LookoutEquipmentModel() *Model { return lookoutEquipmentModel() }
