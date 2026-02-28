package awsmodels

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed redshift/operations.json
var redshiftOperationsJSON []byte

//go:embed redshift/types.json
var redshiftTypesJSON []byte

var (
	redshiftOnce       sync.Once
	redshiftOperations []string
	redshiftTypes      []string
	redshiftOpSet      map[string]struct{}
)

func loadRedshiftModels() {
	redshiftOnce.Do(func() {
		_ = json.Unmarshal(redshiftOperationsJSON, &redshiftOperations)
		_ = json.Unmarshal(redshiftTypesJSON, &redshiftTypes)
		redshiftOpSet = make(map[string]struct{}, len(redshiftOperations))
		for _, op := range redshiftOperations {
			redshiftOpSet[op] = struct{}{}
		}
	})
}

func RedshiftOperations() []string {
	loadRedshiftModels()
	out := make([]string, len(redshiftOperations))
	copy(out, redshiftOperations)
	return out
}

func RedshiftTypes() []string {
	loadRedshiftModels()
	out := make([]string, len(redshiftTypes))
	copy(out, redshiftTypes)
	return out
}

func IsRedshiftOperation(name string) bool {
	loadRedshiftModels()
	_, ok := redshiftOpSet[name]
	return ok
}
