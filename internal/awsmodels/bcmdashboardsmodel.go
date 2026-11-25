package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../bcm-dashboards/2025-08-18/service-2.json internal/awsmodels/bcmdashboards/model.json

//go:embed bcmdashboards/model.json
var bcmDashboardsModelJSON []byte

var bcmDashboardsModel = sync.OnceValue(func() *Model { return mustLoadModel("bcmdashboards", bcmDashboardsModelJSON) })

// BCMDashboardsModel returns the validation model for AWS Billing and Cost Management Dashboards.
func BCMDashboardsModel() *Model { return bcmDashboardsModel() }
