package server

import (
	"bytes"
	"encoding/json"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func TestGeoMapsStage0CatalogMatchesModel(t *testing.T) {
	geoCatalogMatchesModel(t, awsmodels.GeoMapsModel(), geomapsOperationByName, gmapService.handlers)
}

func TestGeoMapsStage0Routing(t *testing.T) {
	ts := newGeoTestServer(t)
	// Unknown geo-maps path.
	geoReject(t, ts, "geo-maps", http.MethodGet, "/v2/nothing", "", "UnknownOperation")
	// The older Location API (signing name geo) keeps its traffic.
	status, _, data := geoCall(t, ts, "geo", http.MethodPost, "/maps/v0/list-maps", "{}")
	if status != http.StatusOK || !strings.Contains(string(data), "Entries") {
		t.Fatalf("location ListMaps: %d %s", status, data)
	}
	// A geo-maps path signed for another service is not claimed.
	if _, _, data := geoCall(t, ts, "s3", http.MethodGet, "/v2/styles/Standard/descriptor", ""); bytes.Contains(data, []byte(`"version":8`)) {
		t.Fatal("an s3-signed request reached geo-maps")
	}
	// Unsigned requests to a geo-maps host need an API key.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v2/styles/Standard/descriptor", nil)
	req.Host = "maps.geo.us-east-1.amazonaws.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden || resp.Header.Get("X-Amzn-ErrorType") != "AccessDeniedException" {
		t.Fatalf("unsigned without key: %d", resp.StatusCode)
	}
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/v2/styles/Standard/descriptor?key=v1.public.test", nil)
	req.Host = "maps.geo.us-east-1.amazonaws.com"
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "maps.geo.us-east-1.amazonaws.com/v2/tiles/vector.basemap") {
		t.Fatalf("API key style: %d %s", resp.StatusCode, body)
	}
}

func TestGeoMapsStage1Payloads(t *testing.T) {
	ts := newGeoTestServer(t)
	get := func(path, wantType string) []byte {
		t.Helper()
		status, h, data := geoCall(t, ts, "geo-maps", http.MethodGet, path, "")
		if status != http.StatusOK || h.Get("Content-Type") != wantType || h.Get("ETag") == "" || h.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("%s: %d %v %s", path, status, h, data)
		}
		return data
	}
	var style map[string]any
	if err := json.Unmarshal(get("/v2/styles/Monochrome/descriptor?color-scheme=Dark&travel-modes=Transit,Truck", "application/json"), &style); err != nil || style["version"] != float64(8) {
		t.Fatalf("style: %v %v", style, err)
	}
	tile := get("/v2/tiles/vector.basemap/3/2/1", "application/vnd.mapbox-vector-tile")
	if !bytes.Contains(tile, []byte("stackyard")) {
		t.Fatalf("tile %x", tile)
	}
	if _, err := png.Decode(bytes.NewReader(get("/v2/tiles/raster.dem/0/0/0", "image/png"))); err != nil {
		t.Fatal(err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(get("/v2/tiles/raster.satellite/1/1/1", "image/jpeg"))); err != nil {
		t.Fatal(err)
	}
	sprite, err := png.Decode(bytes.NewReader(get("/v2/styles/Standard/Light/Default/sprites/sprites@2x.png", "image/png")))
	if err != nil || sprite.Bounds().Dx() != 32 {
		t.Fatalf("sprite: %v", err)
	}
	var index map[string]map[string]any
	if err := json.Unmarshal(get("/v2/styles/Standard/Light/Default/sprites/sprites.json", "application/json"), &index); err != nil || index["stackyard-marker"]["width"] != float64(16) {
		t.Fatalf("sprite index: %v %v", index, err)
	}
	glyphs := get("/v2/glyphs/Amazon%20Ember%20Regular/256-511.pbf", "application/x-protobuf")
	if !bytes.Contains(glyphs, []byte("Amazon Ember Regular")) || !bytes.Contains(glyphs, []byte("256-511")) {
		t.Fatalf("glyphs %q", glyphs)
	}
	img, err := jpeg.Decode(bytes.NewReader(get("/v2/static/map@2x?center=-122.33,47.6&zoom=10&width=100&height=80", "image/jpeg")))
	if err != nil || img.Bounds().Dx() != 200 || img.Bounds().Dy() != 160 {
		t.Fatalf("static map: %v %v", err, img)
	}
}

func TestGeoMapsStage1Validation(t *testing.T) {
	ts := newGeoTestServer(t)
	cases := []struct{ name, path, reason string }{
		{"bad style", "/v2/styles/Neon/descriptor", "FieldValidationFailed"},
		{"bad enum list", "/v2/styles/Standard/descriptor?travel-modes=Boat", "FieldValidationFailed"},
		{"zoom out of range", "/v2/tiles/vector.basemap/23/0/0", "FieldValidationFailed"},
		{"x beyond zoom", "/v2/tiles/vector.basemap/1/2/0", "FieldValidationFailed"},
		{"bad glyph range", "/v2/glyphs/Noto/1-300.pbf", "FieldValidationFailed"},
		{"sprite file name", "/v2/styles/Standard/Light/Default/sprites/icons.png", "FieldValidationFailed"},
		{"static map needs size", "/v2/static/map?center=0,0&zoom=1", "Missing"},
		{"static map needs an anchor", "/v2/static/map?width=64&height=64", "Missing"},
		{"static map anchors exclusive", "/v2/static/map?width=64&height=64&center=0,0&bounding-box=0,0,1,1", "FieldValidationFailed"},
		{"static map latitude", "/v2/static/map?width=64&height=64&center=0,95&zoom=3", "FieldValidationFailed"},
		{"static map too wide", "/v2/static/map?width=5000&height=64&center=0,0&zoom=3", "FieldValidationFailed"},
		{"static map width not a number", "/v2/static/map?width=abc&height=64&center=0,0", "CannotParse"},
		{"bad overlay", "/v2/static/map?width=64&height=64&geojson-overlay=nope", "CannotParse"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			geoReject(t, ts, "geo-maps", http.MethodGet, tc.path, "", tc.reason)
		})
	}
	status, h, _ := geoCall(t, ts, "geo-maps", http.MethodGet, "/v2/tiles/nothing.here/0/0/0", "")
	if status != http.StatusNotFound || h.Get("X-Amzn-ErrorType") != "ResourceNotFoundException" {
		t.Fatalf("unknown tileset: %d", status)
	}
}
