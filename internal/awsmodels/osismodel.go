package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed osis/model.json
var osisModelJSON []byte

var osisModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(osisModelJSON)
	if err != nil {
		panic("awsmodels: embedded osis model is invalid: " + err.Error())
	}
	return m
})

// OSISModel returns the validation model for Amazon OpenSearch Ingestion
// (from the AWS CLI 2.41 model via scripts/gen-aws-model-slim.py).
func OSISModel() *Model { return osisModel() }
