package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed timestreamwrite/model.json
var timestreamWriteModelJSON []byte

//go:embed timestreamquery/model.json
var timestreamQueryModelJSON []byte

var timestreamWriteModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(timestreamWriteModelJSON)
	if err != nil {
		panic("awsmodels: embedded timestream-write model is invalid: " + err.Error())
	}
	return m
})

var timestreamQueryModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(timestreamQueryModelJSON)
	if err != nil {
		panic("awsmodels: embedded timestream-query model is invalid: " + err.Error())
	}
	return m
})

// TimestreamWriteModel returns the validation model for Amazon Timestream Write.
func TimestreamWriteModel() *Model { return timestreamWriteModel() }

// TimestreamQueryModel returns the validation model for Amazon Timestream Query.
func TimestreamQueryModel() *Model { return timestreamQueryModel() }
