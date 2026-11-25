package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed mediastoredata/model.json
var mediastoredataModelJSON []byte

var mediastoredataModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(mediastoredataModelJSON)
	if err != nil {
		panic("awsmodels: embedded mediastoredata model is invalid: " + err.Error())
	}
	return m
})

// MediaStoreDataModel returns the validation model for AWS Elemental MediaStore Data.
func MediaStoreDataModel() *Model { return mediastoredataModel() }
