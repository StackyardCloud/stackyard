package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed polly/model.json
var pollyModelJSON []byte

var pollyModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(pollyModelJSON)
	if err != nil {
		panic("awsmodels: embedded polly model is invalid: " + err.Error())
	}
	return m
})

// PollyModel returns the validation model for Amazon Polly.
func PollyModel() *Model { return pollyModel() }
