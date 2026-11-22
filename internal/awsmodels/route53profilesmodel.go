package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed route53profiles/model.json
var route53profilesModelJSON []byte

var route53profilesModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(route53profilesModelJSON)
	if err != nil {
		panic("awsmodels: embedded route53profiles model is invalid: " + err.Error())
	}
	return m
})

// Route53ProfilesModel returns the validation model for Route 53 Profiles.
func Route53ProfilesModel() *Model { return route53profilesModel() }
