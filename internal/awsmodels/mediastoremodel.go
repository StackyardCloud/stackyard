package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed mediastore/model.json
var mediastoreModelJSON []byte

var mediastoreModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(mediastoreModelJSON)
	if err != nil {
		panic("awsmodels: embedded mediastore model is invalid: " + err.Error())
	}
	return m
})

// MediaStoreModel returns the validation model for AWS Elemental MediaStore.
func MediaStoreModel() *Model { return mediastoreModel() }
