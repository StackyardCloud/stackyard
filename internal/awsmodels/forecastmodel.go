package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copies of the AWS CLI 2.42 models, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../forecast/2018-06-26/service-2.json internal/awsmodels/forecast/model.json
//   gen-aws-model-slim.py .../forecastquery/2018-06-26/service-2.json internal/awsmodels/forecastquery/model.json

//go:embed forecast/model.json
var forecastModelJSON []byte

//go:embed forecastquery/model.json
var forecastQueryModelJSON []byte

var forecastModel = sync.OnceValue(func() *Model { return mustLoadModel("forecast", forecastModelJSON) })

var forecastQueryModel = sync.OnceValue(func() *Model { return mustLoadModel("forecastquery", forecastQueryModelJSON) })

// ForecastModel returns the validation model for Amazon Forecast.
func ForecastModel() *Model { return forecastModel() }

// ForecastQueryModel returns the validation model for Amazon Forecast Query.
func ForecastQueryModel() *Model { return forecastQueryModel() }
