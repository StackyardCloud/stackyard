package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed mediapackagevod/model.json
var mediapackagevodModelJSON []byte

var mediapackagevodModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(mediapackagevodModelJSON)
	if err != nil {
		panic("awsmodels: embedded mediapackagevod model is invalid: " + err.Error())
	}
	return m
})

// MediaPackageVODModel returns the validation model for AWS Elemental MediaPackage VOD.
func MediaPackageVODModel() *Model { return mediapackagevodModel() }
