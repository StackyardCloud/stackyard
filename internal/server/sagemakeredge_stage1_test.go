package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestSageMakerEdgeStage1DevicesAndDeployments(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)
	smOK(t, ts, "CreateDeviceFleet", `{"DeviceFleetName":"fleet","OutputConfig":{"S3OutputLocation":"s3://bucket/edge"},"EnableIotRoleAlias":true}`)
	if status, out := smCall(t, ts, "CreateDeviceFleet", `{"DeviceFleetName":"fleet","OutputConfig":{"S3OutputLocation":"s3://bucket/edge"}}`); status != http.StatusBadRequest || out["__type"] != "ResourceInUse" {
		t.Fatalf("duplicate fleet: %d %v", status, out)
	}
	if out := smOK(t, ts, "DescribeDeviceFleet", `{"DeviceFleetName":"fleet"}`); out["IotRoleAlias"] != "arn:aws:iot:us-east-1:123456789012:rolealias/SageMakerEdge-fleet" {
		t.Fatalf("fleet: %v", out)
	}
	smOK(t, ts, "RegisterDevices", `{"DeviceFleetName":"fleet","Devices":[{"DeviceName":"cam-1","Description":"door"},{"DeviceName":"cam-2"},{"DeviceName":"gate-1"}]}`)
	smOK(t, ts, "CreateDeviceFleet", `{"DeviceFleetName":"other","OutputConfig":{"S3OutputLocation":"s3://bucket/edge"}}`)
	if status, _ := smCall(t, ts, "RegisterDevices", `{"DeviceFleetName":"other","Devices":[{"DeviceName":"cam-1"}]}`); status != http.StatusBadRequest {
		t.Fatalf("device in two fleets: %d", status)
	}
	if status, out := smCall(t, ts, "DeleteDeviceFleet", `{"DeviceFleetName":"fleet"}`); status != http.StatusBadRequest || out["__type"] != "ResourceInUse" {
		t.Fatalf("delete fleet with devices: %d %v", status, out)
	}

	edge := func(path, body string) (int, map[string]any) { return smData(t, ts, http.MethodPost, path, body) }
	if status, out := edge("/GetDeviceRegistration", `{"DeviceName":"cam-1","DeviceFleetName":"fleet"}`); status != http.StatusOK || out["DeviceRegistration"] != "REGISTERED" {
		t.Fatalf("GetDeviceRegistration: %d %v", status, out)
	}
	if status, _ := edge("/GetDeviceRegistration", `{"DeviceName":"cam-1","DeviceFleetName":"other"}`); status != http.StatusBadRequest {
		t.Fatalf("device of another fleet: %d", status)
	}

	smOK(t, ts, "CreateEdgeDeploymentPlan", `{"EdgeDeploymentPlanName":"plan","DeviceFleetName":"fleet",
		"ModelConfigs":[{"ModelHandle":"detector","EdgePackagingJobName":"pkg"}],
		"Stages":[{"StageName":"cams","DeviceSelectionConfig":{"DeviceSubsetType":"NAMECONTAINS","DeviceNameContains":"cam"}}]}`)
	if _, out := edge("/GetDeployments", `{"DeviceName":"cam-1","DeviceFleetName":"fleet"}`); len(out["Deployments"].([]any)) != 0 {
		t.Fatalf("deployments before the stage starts: %v", out)
	}
	smOK(t, ts, "StartEdgeDeploymentStage", `{"EdgeDeploymentPlanName":"plan","StageName":"cams"}`)
	_, out := edge("/GetDeployments", `{"DeviceName":"cam-1","DeviceFleetName":"fleet"}`)
	if got := smJSON(out["Deployments"]); got != `[{"Definitions":[{"ModelHandle":"detector","State":"DEPLOY"}],"DeploymentName":"plan","FailureHandlingPolicy":"ROLLBACK_ON_FAILURE","Type":"Model"}]` {
		t.Fatalf("GetDeployments: %s", got)
	}
	if _, out := edge("/GetDeployments", `{"DeviceName":"gate-1","DeviceFleetName":"fleet"}`); len(out["Deployments"].([]any)) != 0 {
		t.Fatalf("gate-1 is not in the stage: %v", out)
	}
	if status, out := smCall(t, ts, "DeleteEdgeDeploymentPlan", `{"EdgeDeploymentPlanName":"plan"}`); status != http.StatusBadRequest || out["__type"] != "ResourceInUse" {
		t.Fatalf("delete a plan in progress: %d %v", status, out)
	}

	// Heartbeats report results; the stage settles once every device has.
	if status, out := edge("/SendHeartbeat", `{"DeviceName":"cam-1","DeviceFleetName":"fleet","AgentVersion":"1.2",
		"Models":[{"ModelName":"detector","ModelVersion":"1","LatestInference":1700000000}],
		"DeploymentResult":{"DeploymentName":"plan","DeploymentStatus":"SUCCESS","DeploymentModels":[{"ModelHandle":"detector","Status":"SUCCESS"}]}}`); status != http.StatusOK {
		t.Fatalf("SendHeartbeat: %d %v", status, out)
	}
	if status, _ := edge("/SendHeartbeat", `{"DeviceName":"cam-1","DeviceFleetName":"fleet","AgentVersion":"1.2","DeploymentResult":{"DeploymentName":"nope"}}`); status != http.StatusBadRequest {
		t.Fatalf("result for an unknown plan: %d", status)
	}
	if _, out := edge("/GetDeployments", `{"DeviceName":"cam-1","DeviceFleetName":"fleet"}`); len(out["Deployments"].([]any)) != 0 {
		t.Fatalf("deployment after success: %v", out)
	}
	out = smOK(t, ts, "DescribeEdgeDeploymentPlan", `{"EdgeDeploymentPlanName":"plan"}`)
	if out["EdgeDeploymentSuccess"] != 1.0 || out["EdgeDeploymentPending"] != 1.0 || !strings.Contains(smJSON(out["Stages"]), `"StageStatus":"INPROGRESS"`) {
		t.Fatalf("plan after one result: %v", out)
	}
	edge("/SendHeartbeat", `{"DeviceName":"cam-2","DeviceFleetName":"fleet","AgentVersion":"1.2","DeploymentResult":{"DeploymentName":"plan","DeploymentStatus":"FAIL"}}`)
	out = smOK(t, ts, "DescribeEdgeDeploymentPlan", `{"EdgeDeploymentPlanName":"plan"}`)
	if out["EdgeDeploymentFailed"] != 1.0 || !strings.Contains(smJSON(out["Stages"]), `"StageStatus":"FAILED"`) {
		t.Fatalf("plan after all results: %v", out)
	}

	// The control plane shows what the agent reported.
	out = smOK(t, ts, "DescribeDevice", `{"DeviceName":"cam-1","DeviceFleetName":"fleet"}`)
	if out["AgentVersion"] != "1.2" || out["LatestHeartbeat"] == nil || !strings.Contains(smJSON(out["Models"]), `"LatestInference":1700000000`) {
		t.Fatalf("DescribeDevice: %v", out)
	}
	if got := smJSON(smOK(t, ts, "ListDevices", `{"ModelName":"detector"}`)["DeviceSummaries"]); !strings.Contains(got, `"cam-1"`) || strings.Contains(got, `"gate-1"`) {
		t.Fatalf("ListDevices by model: %s", got)
	}
	smOK(t, ts, "UpdateDevices", `{"DeviceFleetName":"fleet","Devices":[{"DeviceName":"gate-1","Description":"main gate"}]}`)
	if out := smOK(t, ts, "DescribeDevice", `{"DeviceName":"gate-1","DeviceFleetName":"fleet"}`); out["Description"] != "main gate" {
		t.Fatalf("UpdateDevices: %v", out)
	}
	smOK(t, ts, "DeregisterDevices", `{"DeviceFleetName":"fleet","DeviceNames":["cam-1","cam-2","gate-1"]}`)
	if status, _ := edge("/GetDeviceRegistration", `{"DeviceName":"cam-1","DeviceFleetName":"fleet"}`); status != http.StatusBadRequest {
		t.Fatalf("deregistered device: %d", status)
	}
	if status, out := smCall(t, ts, "DescribeDevice", `{"DeviceName":"cam-1","DeviceFleetName":"fleet"}`); status != http.StatusBadRequest || out["__type"] != "ResourceNotFound" {
		t.Fatalf("DescribeDevice after deregister: %d %v", status, out)
	}
	// The stage settled, so the plan and then the fleet can be deleted.
	smOK(t, ts, "DeleteEdgeDeploymentPlan", `{"EdgeDeploymentPlanName":"plan"}`)
	smOK(t, ts, "DeleteDeviceFleet", `{"DeviceFleetName":"fleet"}`)
	if got := smJSON(smOK(t, ts, "ListDeviceFleets", `{}`)["DeviceFleetSummaries"]); strings.Contains(got, `"fleet"`) || !strings.Contains(got, `"other"`) {
		t.Fatalf("ListDeviceFleets: %s", got)
	}
}

func TestSageMakerEdgeStage1PlanValidation(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)
	smOK(t, ts, "CreateDeviceFleet", `{"DeviceFleetName":"fleet","OutputConfig":{"S3OutputLocation":"s3://bucket/edge"}}`)
	for name, body := range map[string]string{
		"no fleet":           `{"EdgeDeploymentPlanName":"p","DeviceFleetName":"nope","ModelConfigs":[{"ModelHandle":"m","EdgePackagingJobName":"j"}]}`,
		"no models":          `{"EdgeDeploymentPlanName":"p","DeviceFleetName":"fleet","ModelConfigs":[]}`,
		"percentage":         `{"EdgeDeploymentPlanName":"p","DeviceFleetName":"fleet","ModelConfigs":[{"ModelHandle":"m","EdgePackagingJobName":"j"}],"Stages":[{"StageName":"s","DeviceSelectionConfig":{"DeviceSubsetType":"PERCENTAGE"}}]}`,
		"duplicate stage":    `{"EdgeDeploymentPlanName":"p","DeviceFleetName":"fleet","ModelConfigs":[{"ModelHandle":"m","EdgePackagingJobName":"j"}],"Stages":[{"StageName":"s","DeviceSelectionConfig":{"DeviceSubsetType":"PERCENTAGE","Percentage":5}},{"StageName":"s","DeviceSelectionConfig":{"DeviceSubsetType":"PERCENTAGE","Percentage":5}}]}`,
		"empty selection":    `{"EdgeDeploymentPlanName":"p","DeviceFleetName":"fleet","ModelConfigs":[{"ModelHandle":"m","EdgePackagingJobName":"j"}],"Stages":[{"StageName":"s","DeviceSelectionConfig":{"DeviceSubsetType":"SELECTION"}}]}`,
		"bad subset (model)": `{"EdgeDeploymentPlanName":"p","DeviceFleetName":"fleet","ModelConfigs":[{"ModelHandle":"m","EdgePackagingJobName":"j"}],"Stages":[{"StageName":"s","DeviceSelectionConfig":{"DeviceSubsetType":"ALL"}}]}`,
	} {
		if status, out := smCall(t, ts, "CreateEdgeDeploymentPlan", body); status != http.StatusBadRequest {
			t.Errorf("%s: %d %v", name, status, out)
		}
	}
	smOK(t, ts, "CreateEdgeDeploymentPlan", `{"EdgeDeploymentPlanName":"p","DeviceFleetName":"fleet","ModelConfigs":[{"ModelHandle":"m","EdgePackagingJobName":"j"}],"Stages":[{"StageName":"all","DeviceSelectionConfig":{"DeviceSubsetType":"PERCENTAGE","Percentage":100}}]}`)
	if status, _ := smCall(t, ts, "StopEdgeDeploymentStage", `{"EdgeDeploymentPlanName":"p","StageName":"all"}`); status != http.StatusBadRequest {
		t.Fatalf("stop a stage that never started: %d", status)
	}
	if status, out := smCall(t, ts, "StartEdgeDeploymentStage", `{"EdgeDeploymentPlanName":"p","StageName":"nope"}`); status != http.StatusBadRequest || out["__type"] != "ResourceNotFound" {
		t.Fatalf("start a missing stage: %d %v", status, out)
	}
	smOK(t, ts, "StartEdgeDeploymentStage", `{"EdgeDeploymentPlanName":"p","StageName":"all"}`)
	smOK(t, ts, "StopEdgeDeploymentStage", `{"EdgeDeploymentPlanName":"p","StageName":"all"}`)
	if got := smJSON(smOK(t, ts, "DescribeEdgeDeploymentPlan", `{"EdgeDeploymentPlanName":"p"}`)["Stages"]); !strings.Contains(got, `"StageStatus":"STOPPED"`) {
		t.Fatalf("stopped stage: %s", got)
	}
}
