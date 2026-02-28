package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIoTFleetWiseStage12CatalogAndManifestLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotFleetWiseRequest(t, ts, "CreateSignalCatalog", `{"clientToken":"stage-iotfw-create-sc-token-000001","name":"stage-signal-catalog"}`)
	assertStatus(t, resp, http.StatusOK)
	signalCatalog := decodeIoTFleetWisePayload(t, resp)

	resp = iotFleetWiseRequest(t, ts, "GetSignalCatalog", `{"name":"stage-signal-catalog"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "CreateModelManifest", `{"clientToken":"stage-iotfw-create-mm-token-000001","name":"stage-model-manifest","signalCatalogName":"stage-signal-catalog"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "GetModelManifest", `{"name":"stage-model-manifest"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "CreateDecoderManifest", `{"clientToken":"stage-iotfw-create-dm-token-000001","name":"stage-decoder-manifest","modelManifestName":"stage-model-manifest"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "GetDecoderManifest", `{"name":"stage-decoder-manifest"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "ListSignalCatalogs", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-signal-catalog") {
		t.Fatalf("expected ListSignalCatalogs to include stage-signal-catalog, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "ListModelManifests", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-model-manifest") {
		t.Fatalf("expected ListModelManifests to include stage-model-manifest, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "ListDecoderManifests", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-decoder-manifest") {
		t.Fatalf("expected ListDecoderManifests to include stage-decoder-manifest, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "ListSignalCatalogNodes", `{"name":"stage-signal-catalog"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "ListModelManifestNodes", `{"name":"stage-model-manifest"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "ListDecoderManifestNetworkInterfaces", `{"name":"stage-decoder-manifest"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "ListDecoderManifestSignals", `{"name":"stage-decoder-manifest"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "UpdateSignalCatalog", `{"name":"stage-signal-catalog","description":"updated-signal-catalog"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "UpdateModelManifest", `{"name":"stage-model-manifest","description":"updated-model-manifest"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "UpdateDecoderManifest", `{"name":"stage-decoder-manifest","description":"updated-decoder-manifest"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "ImportSignalCatalog", `{"clientToken":"stage-iotfw-import-sc-token-000001","name":"stage-imported-signal-catalog"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "ImportDecoderManifest", `{"clientToken":"stage-iotfw-import-dm-token-000001","name":"stage-imported-decoder-manifest"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "DeleteDecoderManifest", `{"name":"stage-decoder-manifest"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "DeleteModelManifest", `{"name":"stage-model-manifest"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "DeleteSignalCatalog", `{"name":"stage-signal-catalog"}`)
	assertStatus(t, resp, http.StatusOK)

	if iotFleetWisePayloadString(signalCatalog, "arn") == "" {
		t.Fatalf("expected CreateSignalCatalog response to include arn")
	}
}

func TestIoTFleetWiseStage3FleetVehicleAssociationAndStatus(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotFleetWiseRequest(t, ts, "CreateFleet", `{"clientToken":"stage-iotfw-create-fleet-token-000001","name":"stage-fleet"}`)
	assertStatus(t, resp, http.StatusOK)
	fleet := decodeIoTFleetWisePayload(t, resp)
	fleetID := iotFleetWisePayloadString(fleet, "id")
	if fleetID == "" {
		t.Fatalf("expected CreateFleet to return id")
	}

	resp = iotFleetWiseRequest(t, ts, "CreateVehicle", `{"clientToken":"stage-iotfw-create-vehicle-token-000001","vehicleName":"stage-vehicle","fleetId":"`+fleetID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "AssociateVehicleFleet", `{"vehicleName":"stage-vehicle","fleetId":"`+fleetID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "ListFleetsForVehicle", `{"vehicleName":"stage-vehicle"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-fleet") {
		t.Fatalf("expected ListFleetsForVehicle to include stage-fleet, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "ListVehiclesInFleet", `{"fleetId":"`+fleetID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-vehicle") {
		t.Fatalf("expected ListVehiclesInFleet to include stage-vehicle, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "GetVehicleStatus", `{"vehicleName":"stage-vehicle"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "campaigns") {
		t.Fatalf("expected GetVehicleStatus to include campaigns, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "UpdateFleet", `{"fleetId":"`+fleetID+`","description":"updated-stage-fleet"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "UpdateVehicle", `{"vehicleName":"stage-vehicle","description":"updated-stage-vehicle"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "DisassociateVehicleFleet", `{"vehicleName":"stage-vehicle","fleetId":"`+fleetID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "DeleteVehicle", `{"vehicleName":"stage-vehicle"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "DeleteFleet", `{"fleetId":"`+fleetID+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestIoTFleetWiseStage45CampaignStateTemplateTaggingAndSettings(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotFleetWiseRequest(t, ts, "RegisterAccount", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "GetRegisterAccountStatus", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "PutLoggingOptions", `{"cloudWatchLogDeliveryOptions":{"logType":"OFF"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "GetLoggingOptions", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "PutEncryptionConfiguration", `{"encryptionStatus":"ENABLED","kmsKeyId":"alias/stackyard-iotfleetwise"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "GetEncryptionConfiguration", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "CreateCampaign", `{"clientToken":"stage-iotfw-create-campaign-token-000001","name":"stage-campaign"}`)
	assertStatus(t, resp, http.StatusOK)
	campaign := decodeIoTFleetWisePayload(t, resp)
	campaignARN := iotFleetWisePayloadString(campaign, "arn")
	if campaignARN == "" {
		t.Fatalf("expected CreateCampaign to return arn")
	}

	resp = iotFleetWiseRequest(t, ts, "CreateStateTemplate", `{"clientToken":"stage-iotfw-create-state-template-token-000001","name":"stage-state-template"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "ListCampaigns", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-campaign") {
		t.Fatalf("expected ListCampaigns to include stage-campaign, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "ListStateTemplates", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-state-template") {
		t.Fatalf("expected ListStateTemplates to include stage-state-template, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "TagResource", `{"resourceArn":"`+campaignARN+`","tags":[{"Key":"env","Value":"stage"},{"Key":"owner","Value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "ListTagsForResource", `{"resourceArn":"`+campaignARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "\"owner\"") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "UntagResource", `{"resourceArn":"`+campaignARN+`","tagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "ListTagsForResource", `{"resourceArn":"`+campaignARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, "\"owner\"") {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "UpdateCampaign", `{"name":"stage-campaign","description":"updated-stage-campaign"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "UpdateStateTemplate", `{"name":"stage-state-template","description":"updated-stage-state-template"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "GetCampaign", `{"name":"stage-campaign"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "GetStateTemplate", `{"name":"stage-state-template"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotFleetWiseRequest(t, ts, "DeleteStateTemplate", `{"name":"stage-state-template"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotFleetWiseRequest(t, ts, "DeleteCampaign", `{"name":"stage-campaign"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestIoTFleetWiseStage6ValidationIdempotencyAndBatchSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := "stage-iotfw-idempotent-create-fleet-token-000001"
	resp := iotFleetWiseRequest(t, ts, "CreateFleet", `{"clientToken":"`+token+`","name":"stage-idempotent-fleet"}`)
	assertStatus(t, resp, http.StatusOK)
	first := decodeIoTFleetWisePayload(t, resp)
	firstID := iotFleetWisePayloadString(first, "id")
	if firstID == "" {
		t.Fatalf("expected first CreateFleet response to include id")
	}

	resp = iotFleetWiseRequest(t, ts, "CreateFleet", `{"clientToken":"`+token+`","name":"ignored-name-due-to-idempotency"}`)
	assertStatus(t, resp, http.StatusOK)
	second := decodeIoTFleetWisePayload(t, resp)
	secondID := iotFleetWisePayloadString(second, "id")
	if firstID != secondID {
		t.Fatalf("expected idempotent CreateFleet to return same id: %s != %s", firstID, secondID)
	}

	resp = iotFleetWiseRequest(t, ts, "BatchCreateVehicle", `{"vehicles":[{"vehicleName":"stage-batch-vehicle-1"},{"vehicleName":"stage-batch-vehicle-2"}]}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-batch-vehicle-1") {
		t.Fatalf("expected BatchCreateVehicle to include stage-batch-vehicle-1, got %q", body)
	}

	resp = iotFleetWiseRequest(t, ts, "BatchUpdateVehicle", `{"vehicles":[{"vehicleName":"stage-batch-vehicle-1","description":"updated"},{"vehicleName":"stage-batch-vehicle-2","description":"updated"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "IoTAutobahnControlPlane.ListFleets",
		},
		"iotfleetwise",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeIoTFleetWisePayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}
