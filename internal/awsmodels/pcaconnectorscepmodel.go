package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed pcaconnectorscep/model.json
var pcaconnectorscepModelJSON []byte

var pcaconnectorscepModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(pcaconnectorscepModelJSON)
	if err != nil {
		panic("awsmodels: embedded pcaconnectorscep model is invalid: " + err.Error())
	}
	return m
})

// PCAConnectorSCEPModel returns the validation model for Private CA Connector for SCEP.
func PCAConnectorSCEPModel() *Model { return pcaconnectorscepModel() }
