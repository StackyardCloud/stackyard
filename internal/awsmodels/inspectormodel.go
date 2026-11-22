package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed inspector/model.json
var inspectorModelJSON []byte

var inspectorModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(inspectorModelJSON)
	if err != nil {
		panic("awsmodels: embedded inspector model is invalid: " + err.Error())
	}
	return m
})

// InspectorModel returns the validation model for Amazon Inspector Classic.
func InspectorModel() *Model { return inspectorModel() }
