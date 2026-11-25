package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed geoplaces/model.json
var geoplacesModelJSON []byte

var geoplacesModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(geoplacesModelJSON)
	if err != nil {
		panic("awsmodels: embedded geoplaces model is invalid: " + err.Error())
	}
	return m
})

// GeoPlacesModel returns the validation model for Amazon Location Service Places V2 (geo-places, CLI 2.42.1).
func GeoPlacesModel() *Model { return geoplacesModel() }
