package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed databrew/model.json
var databrewModelJSON []byte

var databrewModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(databrewModelJSON)
	if err != nil {
		panic("awsmodels: embedded databrew model is invalid: " + err.Error())
	}
	return m
})

// DataBrewModel returns the validation model for AWS Glue DataBrew (from the
// AWS CLI 2.41 model via scripts/gen-aws-model-slim.py).
func DataBrewModel() *Model { return databrewModel() }
