package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestLocationStage12ResourceLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := locationRequest(t, ts, http.MethodPost, "/maps/v0/maps", []byte(`{"MapName":"stage-location-map"}`))
	assertStatus(t, resp, http.StatusOK)
	mapPayload := decodeLocationPayload(t, resp)
	if locationPayloadStringValue(mapPayload, "MapArn") == "" {
		t.Fatalf("expected CreateMap to return MapArn")
	}

	resp = locationRequest(t, ts, http.MethodPost, "/places/v0/indexes", []byte(`{"IndexName":"stage-location-index"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/routes/v0/calculators", []byte(`{"CalculatorName":"stage-location-calculator"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/tracking/v0/trackers", []byte(`{"TrackerName":"stage-location-tracker"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/geofencing/v0/collections", []byte(`{"CollectionName":"stage-location-collection"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/metadata/v0/keys", []byte(`{"KeyName":"stage-location-key"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = locationRequest(t, ts, http.MethodGet, "/maps/v0/maps/stage-location-map", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/places/v0/indexes/stage-location-index", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/routes/v0/calculators/stage-location-calculator", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/tracking/v0/trackers/stage-location-tracker", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/geofencing/v0/collections/stage-location-collection", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/metadata/v0/keys/stage-location-key", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = locationRequest(t, ts, http.MethodPost, "/maps/v0/list-maps", []byte(`{"MaxResults":20}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "Entries") {
		t.Fatalf("expected ListMaps to include Entries, got %q", body)
	}

	resp = locationRequest(t, ts, http.MethodPatch, "/maps/v0/maps/stage-location-map", []byte(`{"Description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPatch, "/places/v0/indexes/stage-location-index", []byte(`{"Description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPatch, "/routes/v0/calculators/stage-location-calculator", []byte(`{"Description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPatch, "/tracking/v0/trackers/stage-location-tracker", []byte(`{"Description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPatch, "/geofencing/v0/collections/stage-location-collection", []byte(`{"Description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPatch, "/metadata/v0/keys/stage-location-key", []byte(`{"Description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = locationRequest(t, ts, http.MethodDelete, "/metadata/v0/keys/stage-location-key", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodDelete, "/geofencing/v0/collections/stage-location-collection", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodDelete, "/tracking/v0/trackers/stage-location-tracker", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodDelete, "/routes/v0/calculators/stage-location-calculator", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodDelete, "/places/v0/indexes/stage-location-index", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodDelete, "/maps/v0/maps/stage-location-map", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestLocationStage34TrackingAndGeofencingSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	tracker := "stackyard-tracker"
	collection := "stackyard-geofence-collection"

	resp := locationRequest(t, ts, http.MethodPost, "/tracking/v0/trackers/"+tracker+"/positions", []byte(`{"Updates":[{"DeviceId":"device-stage","Position":[-122.1,47.6]}]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/tracking/v0/trackers/"+tracker+"/get-positions", []byte(`{"DeviceIds":["device-stage"]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/tracking/v0/trackers/"+tracker+"/devices/device-stage/positions/latest", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/tracking/v0/trackers/"+tracker+"/devices/device-stage/list-positions", []byte(`{"MaxResults":20}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/tracking/v0/trackers/"+tracker+"/list-positions", []byte(`{"MaxResults":20}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/tracking/v0/trackers/"+tracker+"/positions/verify", []byte(`{"DeviceState":{"Position":[-122.1,47.6],"SampleTime":"2026-01-01T00:00:00Z"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/tracking/v0/trackers/"+tracker+"/delete-positions", []byte(`{"DeviceIds":["device-stage"]}`))
	assertStatus(t, resp, http.StatusOK)

	consumerARN := url.PathEscape("arn:aws:kinesis:us-east-1:123456789012:stream/location-consumer")
	resp = locationRequest(t, ts, http.MethodPost, "/tracking/v0/trackers/"+tracker+"/consumers", []byte(`{"ConsumerArn":"arn:aws:kinesis:us-east-1:123456789012:stream/location-consumer"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/tracking/v0/trackers/"+tracker+"/list-consumers", []byte(`{"MaxResults":20}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodDelete, "/tracking/v0/trackers/"+tracker+"/consumers/"+consumerARN, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = locationRequest(t, ts, http.MethodPut, "/geofencing/v0/collections/"+collection+"/geofences/stage-geofence", []byte(`{"GeofenceGeometry":{"Circle":{"Center":[-122.1,47.6],"Radius":50}}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/geofencing/v0/collections/"+collection+"/geofences/stage-geofence", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/geofencing/v0/collections/"+collection+"/list-geofences", []byte(`{"MaxResults":20}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/geofencing/v0/collections/"+collection+"/put-geofences", []byte(`{"Entries":[{"GeofenceId":"stage-geofence-2","GeofenceGeometry":{"Circle":{"Center":[-122.0,47.7],"Radius":75}}}]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/geofencing/v0/collections/"+collection+"/delete-geofences", []byte(`{"GeofenceIds":["stage-geofence-2"]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/geofencing/v0/collections/"+collection+"/positions", []byte(`{"DevicePositionUpdates":[{"DeviceId":"device-stage","Position":[-122.1,47.6],"SampleTime":"2026-01-01T00:00:00Z"}]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/geofencing/v0/collections/"+collection+"/forecast-geofence-events", []byte(`{"DeviceState":{"DeviceId":"device-stage","Position":[-122.1,47.6],"SampleTime":"2026-01-01T00:00:00Z"}}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLocationStage56RouteSearchTaggingAndValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	calculator := "stackyard-route-calculator"
	index := "stackyard-place-index"
	mapName := "stackyard-map"
	resourceARN := url.PathEscape("arn:aws:geo:us-east-1:123456789012:map/stackyard-map")

	resp := locationRequest(t, ts, http.MethodPost, "/routes/v0/calculators/"+calculator+"/calculate/route", []byte(`{"DeparturePosition":[-122.3,47.6],"DestinationPosition":[-122.2,47.7]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/routes/v0/calculators/"+calculator+"/calculate/route-matrix", []byte(`{"DeparturePositions":[[-122.3,47.6]],"DestinationPositions":[[-122.2,47.7]]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = locationRequest(t, ts, http.MethodPost, "/places/v0/indexes/"+index+"/search/text", []byte(`{"Text":"Seattle"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/places/v0/indexes/"+index+"/search/position", []byte(`{"Position":[-122.3,47.6]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodPost, "/places/v0/indexes/"+index+"/search/suggestions", []byte(`{"Text":"Sea"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/places/v0/indexes/"+index+"/places/place-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = locationRequest(t, ts, http.MethodGet, "/maps/v0/maps/"+mapName+"/style-descriptor", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/maps/v0/maps/"+mapName+"/tiles/0/0/0", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/maps/v0/maps/"+mapName+"/sprites/sprite@2x.png", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/maps/v0/maps/"+mapName+"/glyphs/Arial/0-255.pbf", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = locationRequest(t, ts, http.MethodPost, "/tags/"+resourceARN, []byte(`{"Tags":{"owner":"qa","env":"stage"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = locationRequest(t, ts, http.MethodGet, "/tags/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = locationRequest(t, ts, http.MethodDelete, "/tags/"+resourceARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = locationRequest(t, ts, http.MethodPost, "/location/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/maps/v0/list-maps",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"geo",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeLocationPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func locationPayloadStringValue(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
