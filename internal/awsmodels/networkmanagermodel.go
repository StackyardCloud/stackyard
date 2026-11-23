package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed networkmanager/model.json
var networkmanagerModelJSON []byte

var networkmanagerModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(networkmanagerModelJSON)
	if err != nil {
		panic("awsmodels: embedded networkmanager model is invalid: " + err.Error())
	}
	return m
})

// NetworkManagerModel returns the validation model for AWS Network Manager.
func NetworkManagerModel() *Model { return networkmanagerModel() }
