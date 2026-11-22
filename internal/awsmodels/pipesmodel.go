package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed pipes/model.json
var pipesModelJSON []byte

var pipesModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(pipesModelJSON)
	if err != nil {
		panic("awsmodels: embedded pipes model is invalid: " + err.Error())
	}
	return m
})

// PipesModel returns the validation model for Amazon EventBridge Pipes.
func PipesModel() *Model { return pipesModel() }
