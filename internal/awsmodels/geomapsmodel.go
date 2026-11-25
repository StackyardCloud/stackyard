package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed geomaps/model.json
var geomapsModelJSON []byte

var geomapsModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(geomapsModelJSON)
	if err != nil {
		panic("awsmodels: embedded geomaps model is invalid: " + err.Error())
	}
	return m
})

// GeoMapsModel returns the validation model for Amazon Location Service Maps V2 (geo-maps, CLI 2.42.1).
func GeoMapsModel() *Model { return geomapsModel() }
