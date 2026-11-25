package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed georoutes/model.json
var georoutesModelJSON []byte

var georoutesModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(georoutesModelJSON)
	if err != nil {
		panic("awsmodels: embedded georoutes model is invalid: " + err.Error())
	}
	return m
})

// GeoRoutesModel returns the validation model for Amazon Location Service Routes V2 (geo-routes, CLI 2.42.1).
func GeoRoutesModel() *Model { return georoutesModel() }
