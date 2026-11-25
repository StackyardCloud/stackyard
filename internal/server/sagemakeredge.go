package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon SageMaker Edge Manager agent API (edge.sagemaker, REST JSON, signed
// "sagemaker"). Devices and fleets are the ones registered with the SageMaker
// control plane (sagemaker_edge.go); deployments come from the in-progress
// stages of its edge deployment plans, and heartbeats update the device and
// the plan's per-device deployment results.
//
// API reference:
// https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_Operations_Amazon_Sagemaker_Edge_Manager.html

const (
	smeMaxBody = 1 << 20
	smeHost    = "edge.sagemaker."
	// smeMaxHeartbeatModels caps the models a device reports.
	smeMaxHeartbeatModels = 50
	smeRegistrationTTL    = "3600"
)

var smePaths = map[string]string{
	"/GetDeployments": "GetDeployments", "/GetDeviceRegistration": "GetDeviceRegistration", "/SendHeartbeat": "SendHeartbeat",
}

func (s *Server) handleSageMakerEdgeRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost || !sagemakerDataPlaneClaims(r, smeHost) {
		return false
	}
	op, ok := smePaths[rawRequestPath(r)]
	if !ok {
		return false
	}
	body, ok := s.runSageMakerDataPlane(w, r, awsmodels.SageMakerEdgeModel(), op, smeMaxBody, func(raw []byte) (any, *sagemakerDataError) {
		var req smeAgentReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, smfsInvalid("invalid request: %v", err)
		}
		st := s.sagemaker
		st.mu.Lock()
		defer st.mu.Unlock()
		st.initResourcesLocked()
		d, err := st.smeDeviceLocked(req.DeviceName, req.DeviceFleetName)
		if err != nil {
			return nil, smfsInvalid("%s", err.msg)
		}
		switch op {
		case "GetDeviceRegistration":
			return map[string]any{"DeviceRegistration": "REGISTERED", "CacheTTL": smeRegistrationTTL}, nil
		case "GetDeployments":
			return st.smeDeploymentsLocked(d), nil
		}
		return nil, st.smeHeartbeatLocked(d, req, time.Now().UTC())
	})
	if ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
	return true
}

type smeAgentReq struct {
	DeviceName      string
	DeviceFleetName string
	AgentVersion    string
	Models          []struct {
		ModelName        string
		ModelVersion     string
		LatestSampleTime *sagemakerTimestamp
		LatestInference  *sagemakerTimestamp
	}
	DeploymentResult *struct {
		DeploymentName   string
		DeploymentStatus string
		DeploymentModels []struct{ Status string }
	}
}

// smeDeploymentsLocked lists, per plan of the device's fleet, the first
// in-progress stage that targets the device and has no result from it yet.
// Work is bounded by smeMaxPlans * smeMaxStages.
func (s *sagemakerStore) smeDeploymentsLocked(d *smeDevice) any {
	deployments := []any{}
	for _, name := range sagemakerSortNames(s.edgePlans, false, false, nil) {
		p := s.edgePlans[name]
		if p.Fleet != d.Fleet || p.results[d.Name] != "" {
			continue
		}
		for _, stage := range p.Stages {
			if stage.status != "INPROGRESS" || !stage.DeviceSelectionConfig.selects(d.Name) {
				continue
			}
			defs := make([]any, 0, len(p.ModelConfigs))
			for _, mc := range p.ModelConfigs {
				defs = append(defs, map[string]any{"ModelHandle": mc.ModelHandle, "State": "DEPLOY"})
			}
			deployments = append(deployments, map[string]any{"DeploymentName": p.Name, "Type": "Model",
				"FailureHandlingPolicy": stage.DeploymentConfig.FailureHandlingPolicy, "Definitions": defs})
			break
		}
	}
	return map[string]any{"Deployments": deployments}
}

func (s *sagemakerStore) smeHeartbeatLocked(d *smeDevice, req smeAgentReq, now time.Time) *sagemakerDataError {
	if len(req.Models) > smeMaxHeartbeatModels {
		return smfsInvalid("A heartbeat reports at most %d models.", smeMaxHeartbeatModels)
	}
	var plan *smePlan
	result := "SUCCESS"
	if dr := req.DeploymentResult; dr != nil {
		if plan = s.edgePlans[dr.DeploymentName]; plan == nil || plan.Fleet != d.Fleet {
			return smfsInvalid("Deployment %s is not an edge deployment plan of device fleet %s.", dr.DeploymentName, d.Fleet)
		}
		if dr.DeploymentStatus == "FAIL" || slices.ContainsFunc(dr.DeploymentModels, func(m struct{ Status string }) bool { return m.Status == "FAIL" }) {
			result = "FAIL"
		}
	}
	models := make([]smeEdgeModel, 0, len(req.Models))
	for _, m := range req.Models {
		em := smeEdgeModel{ModelName: m.ModelName, ModelVersion: m.ModelVersion}
		if m.LatestSampleTime != nil {
			v := sagemakerEpoch(m.LatestSampleTime.Time)
			em.LatestSampleTime = &v
		}
		if m.LatestInference != nil {
			v := sagemakerEpoch(m.LatestInference.Time)
			em.LatestInference = &v
		}
		models = append(models, em)
	}
	// ponytail: AgentMetrics and model metrics are accepted but not stored;
	// nothing in the emulator reads them.
	d.AgentVersion, d.Heartbeat, d.Models = req.AgentVersion, now, models
	if plan != nil {
		plan.results[d.Name] = result
	}
	return nil
}
