package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func TestGeoRoutesStage0CatalogMatchesModel(t *testing.T) {
	geoCatalogMatchesModel(t, awsmodels.GeoRoutesModel(), georoutesOperationByName, grteService.handlers)
}

func TestGeoRoutesStage0Routing(t *testing.T) {
	ts := newGeoTestServer(t)
	geoReject(t, ts, "geo-routes", http.MethodGet, "/v2/routes", "", "UnknownOperation")
	// Location's route calculators (signing name geo) keep their traffic.
	status, _, data := geoCall(t, ts, "geo", http.MethodPost, "/routes/v0/calculators/calc/calculate/route",
		`{"DeparturePosition":[-122.33,47.6],"DestinationPosition":[-122.34,47.62]}`)
	if status >= 500 || strings.Contains(string(data), "LegGeometryFormat") {
		t.Fatalf("location CalculateRoute: %d %s", status, data)
	}
}

func TestGeoRoutesStage1Routes(t *testing.T) {
	ts := newGeoTestServer(t)
	md := awsmodels.GeoRoutesModel()
	call := func(op, path, body string) map[string]any {
		t.Helper()
		return geoJSON(t, ts, md, "geo-routes", op, http.MethodPost, path, body)
	}

	out := call("CalculateRoutes", "/v2/routes", `{"Origin":[-122.33207,47.60621],"Destination":[-123.11934,49.24966],
		"Waypoints":[{"Position":[-122.5,48.0],"PassThrough":true},{"Position":[-122.6,48.5],"StopDuration":600}],
		"LegGeometryFormat":"Simple","DepartureTime":"2026-01-02T08:00:00-08:00"}`)
	route := out["Routes"].([]any)[0].(map[string]any)
	legs := route["Legs"].([]any)
	if len(legs) != 2 {
		t.Fatalf("legs: %v", legs)
	}
	first := legs[0].(map[string]any)
	if len(first["Geometry"].(map[string]any)["LineString"].([]any)) != 3 {
		t.Fatalf("pass-through waypoint not in the first leg: %v", first)
	}
	details := first["VehicleLegDetails"].(map[string]any)
	if len(details["PassThroughWaypoints"].([]any)) != 1 || details["Departure"].(map[string]any)["Time"] != "2026-01-02T08:00:00-08:00" {
		t.Fatalf("leg details: %v", details)
	}
	summary := route["Summary"].(map[string]any)
	if d := summary["Distance"].(float64); d < 190000 || d > 230000 {
		t.Fatalf("distance %v", d)
	}

	out = call("CalculateRoutes", "/v2/routes", `{"Origin":[0,0],"Destination":[0,0.01],"TravelMode":"Pedestrian"}`)
	leg := out["Routes"].([]any)[0].(map[string]any)["Legs"].([]any)[0].(map[string]any)
	if leg["Type"] != "Pedestrian" || leg["Geometry"].(map[string]any)["Polyline"] == nil || out["LegGeometryFormat"] != "FlexiblePolyline" {
		t.Fatalf("pedestrian leg: %v", leg)
	}

	out = call("CalculateRouteMatrix", "/v2/route-matrix", `{"Origins":[{"Position":[0,0]},{"Position":[10,10]}],
		"Destinations":[{"Position":[0,0.1]}],"RoutingBoundary":{"Geometry":{"Circle":{"Center":[0,0],"Radius":50000}}}}`)
	matrix := out["RouteMatrix"].([]any)
	if out["ErrorCount"] != float64(1) || matrix[1].([]any)[0].(map[string]any)["Error"] != "OutOfBoundsOrigin" {
		t.Fatalf("matrix: %v", out)
	}
	if d := matrix[0].([]any)[0].(map[string]any)["Distance"].(float64); d < 11000 || d > 11200 {
		t.Fatalf("matrix distance %v", d)
	}
	out = call("CalculateRouteMatrix", "/v2/route-matrix", `{"Origins":[{"Position":[0,0]}],"Destinations":[{"Position":[0,1]}],
		"RoutingBoundary":{"Geometry":{"AutoCircle":{"Margin":1000}}}}`)
	if out["RoutingBoundary"].(map[string]any)["Geometry"].(map[string]any)["Circle"] == nil || out["ErrorCount"] != float64(0) {
		t.Fatalf("auto circle: %v", out)
	}
	out = call("CalculateRouteMatrix", "/v2/route-matrix", `{"Origins":[{"Position":[0.5,0.5]}],"Destinations":[{"Position":[0.2,0.8]},{"Position":[2,2]}],
		"RoutingBoundary":{"Geometry":{"Polygon":[[[0,0],[1,0],[1,1],[0,1],[0,0]]]}}}`)
	row := out["RouteMatrix"].([]any)[0].([]any)
	if row[0].(map[string]any)["Error"] != nil || row[1].(map[string]any)["Error"] != "OutOfBoundsDestination" {
		t.Fatalf("polygon boundary: %v", out)
	}

	out = call("CalculateIsolines", "/v2/isolines", `{"Origin":[0,0],"Thresholds":{"Distance":[1000],"Time":[60]},"IsolineGeometryFormat":"Simple"}`)
	isolines := out["Isolines"].([]any)
	ring := isolines[0].(map[string]any)["Geometries"].([]any)[0].(map[string]any)["Polygon"].([]any)[0].([]any)
	if len(isolines) != 2 || len(ring) != grteDefaultRingPoints+1 {
		t.Fatalf("isolines: %v", out)
	}

	out = call("OptimizeWaypoints", "/v2/optimize-waypoints", `{"Origin":[0,0],"Destination":[0,3],"DepartureTime":"2026-01-01T00:00:00Z",
		"Waypoints":[{"Id":"far","Position":[0,2]},{"Id":"near","Position":[0,1],"ServiceDuration":60},{"Id":"first","Position":[0,2.5],"Before":[0]}]}`)
	var order []string
	for _, w := range out["OptimizedWaypoints"].([]any) {
		order = append(order, w.(map[string]any)["Id"].(string))
	}
	if strings.Join(order, ",") != "Origin,near,first,far,Destination" {
		t.Fatalf("optimized order %v", order)
	}
	if out["TimeBreakdown"].(map[string]any)["ServiceDuration"] != float64(60) {
		t.Fatalf("breakdown %v", out["TimeBreakdown"])
	}
	out = call("OptimizeWaypoints", "/v2/optimize-waypoints", `{"Origin":[0,0],"Waypoints":[{"Id":"a","Position":[0,1],"Before":[1]},{"Id":"b","Position":[0,2],"Before":[0]}]}`)
	if len(out["ImpedingWaypoints"].([]any)) != 2 {
		t.Fatalf("cycle: %v", out)
	}

	out = call("SnapToRoads", "/v2/snap-to-roads", `{"TracePoints":[{"Position":[0,0],"Timestamp":"2026-01-01T00:00:10Z"},{"Position":[0,0.001],"Timestamp":"2026-01-01T00:00:05Z"}]}`)
	if len(out["SnappedTracePoints"].([]any)) != 2 || len(out["Notices"].([]any)) != 1 {
		t.Fatalf("snap: %v", out)
	}
}

func TestGeoRoutesStage1Validation(t *testing.T) {
	ts := newGeoTestServer(t)
	many := func(n int, item string) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = item
		}
		return "[" + strings.Join(parts, ",") + "]"
	}
	cases := []struct{ name, path, body, reason string }{
		{"missing destination", "/v2/routes", `{"Origin":[0,0]}`, "Missing"},
		{"origin latitude", "/v2/routes", `{"Origin":[0,100],"Destination":[0,0]}`, "FieldValidationFailed"},
		{"too many waypoints", "/v2/routes", fmt.Sprintf(`{"Origin":[0,0],"Destination":[0,1],"Waypoints":%s}`, many(grteMaxWaypoints+1, `{"Position":[0,0.5]}`)), "FieldValidationFailed"},
		{"bad time", "/v2/routes", `{"Origin":[0,0],"Destination":[0,1],"DepartureTime":"tomorrow"}`, "FieldValidationFailed"},
		{"matrix too big", "/v2/route-matrix", fmt.Sprintf(`{"Origins":%s,"Destinations":[{"Position":[0,0]}]}`, many(grteMaxMatrixSide+1, `{"Position":[0,0]}`)), "FieldValidationFailed"},
		{"boundary both", "/v2/route-matrix", `{"Origins":[{"Position":[0,0]}],"Destinations":[{"Position":[0,0]}],"RoutingBoundary":{"Unbounded":true,"Geometry":{"Circle":{"Center":[0,0],"Radius":5}}}}`, "FieldValidationFailed"},
		{"null polygon", "/v2/route-matrix", `{"Origins":[{"Position":[0,0]}],"Destinations":[{"Position":[0,0]}],"RoutingBoundary":{"Geometry":{"Polygon":null}}}`, "FieldValidationFailed"},
		{"open polygon", "/v2/route-matrix", `{"Origins":[{"Position":[0,0]}],"Destinations":[{"Position":[0,0]}],"RoutingBoundary":{"Geometry":{"Polygon":[[[0,0],[1,0],[1,1],[0,1]]]}}}`, "FieldValidationFailed"},
		{"isoline origin and destination", "/v2/isolines", `{"Origin":[0,0],"Destination":[0,0],"Thresholds":{"Distance":[10]}}`, "FieldValidationFailed"},
		{"isoline without thresholds", "/v2/isolines", `{"Origin":[0,0],"Thresholds":{}}`, "Missing"},
		{"isoline threshold range", "/v2/isolines", `{"Origin":[0,0],"Thresholds":{"Time":[20000]}}`, "FieldValidationFailed"},
		{"optimize too many", "/v2/optimize-waypoints", fmt.Sprintf(`{"Origin":[0,0],"Waypoints":%s}`, many(grteMaxOptimizeStops+1, `{"Position":[0,0]}`)), "FieldValidationFailed"},
		{"optimize bad before", "/v2/optimize-waypoints", `{"Origin":[0,0],"Waypoints":[{"Position":[0,1],"Before":[5]}]}`, "FieldValidationFailed"},
		{"optimize duplicate id", "/v2/optimize-waypoints", `{"Origin":[0,0],"Waypoints":[{"Id":"a","Position":[0,1]},{"Id":"a","Position":[0,2]}]}`, "FieldValidationFailed"},
		{"snap needs two points", "/v2/snap-to-roads", `{"TracePoints":[{"Position":[0,0]}]}`, "FieldValidationFailed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			geoReject(t, ts, "geo-routes", http.MethodPost, tc.path, tc.body, tc.reason)
		})
	}
	if status, _, _ := geoCall(t, ts, "geo-routes", http.MethodPost, "/v2/routes", `{"Origin":[0,0],"Destination":[0,1],"TravelMode":"Boat"}`); status != http.StatusBadRequest {
		t.Fatalf("unknown travel mode: %d", status)
	}
	// Bodies need a declared length within the limit, even when valid JSON.
	valid := `{"TracePoints":[{"Position":[0,0]},{"Position":[0,1]}]}`
	geoReject(t, ts, "geo-routes", http.MethodPost, "/v2/snap-to-roads", valid+strings.Repeat(" ", geosaMaxBody), "Other")
}
