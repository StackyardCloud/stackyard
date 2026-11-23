package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed schemas/model.json
var schemasModelJSON []byte

var schemasModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(schemasModelJSON)
	if err != nil {
		panic("awsmodels: embedded schemas model is invalid: " + err.Error())
	}
	return m
})

// SchemasModel returns the validation model for EventBridge Schemas.
func SchemasModel() *Model { return schemasModel() }
