package server

import (
	"bytes"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	sgeoTestRole     = "arn:aws:iam::123456789012:role/geo"
	sgeoTestSentinel = "arn:aws:sagemaker-geospatial:us-west-2:378778860802:raster-data-collection/public/nmqj48dcu3g7ayw8"
)

func sgeoAwait(t *testing.T, ts *httptest.Server, path, key, want string) map[string]any {
	t.Helper()
	for i := 0; i < 4; i++ {
		out := sgeoOK(t, ts, http.MethodGet, path, "")
		if out[key] == want {
			return out
		}
	}
	t.Fatalf("%s never reached %s=%s", path, key, want)
	return nil
}

func TestSageMakerGeospatialEarthObservation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	if _, err := srv.s3.CreateBucket("geo-out"); err != nil {
		t.Fatal(err)
	}
	query := `"InputConfig":{"RasterDataCollectionQuery":{"RasterDataCollectionArn":"` + sgeoTestSentinel + `","TimeRangeFilter":{"StartTime":1672531200,"EndTime":1675209600},"AreaOfInterest":{"AreaOfInterestGeometry":{"PolygonGeometry":{"Coordinates":[[[-122.4,47.5],[-122.2,47.5],[-122.2,47.7],[-122.4,47.7],[-122.4,47.5]]]}}}}}`

	// Search pages over one scene per revisit and honours property filters.
	search := `{"Arn":"` + sgeoTestSentinel + `","RasterDataCollectionQuery":{"TimeRangeFilter":{"StartTime":1672531200,"EndTime":1704067200},"BandFilter":["red"]}}`
	res := sgeoOK(t, ts, http.MethodPost, "/search-raster-data-collection", search)
	if res["ApproximateResultCount"] != float64(74) || len(res["Items"].([]any)) != sgeoPageSize || res["NextToken"] == nil {
		t.Fatalf("search: count %v next %v", res["ApproximateResultCount"], res["NextToken"])
	}
	if assets := res["Items"].([]any)[0].(map[string]any)["Assets"].(map[string]any); len(assets) != 1 || assets["red"] == nil {
		t.Fatalf("band filter: %v", assets)
	}
	filtered := sgeoOK(t, ts, http.MethodPost, "/search-raster-data-collection", strings.Replace(search, `"BandFilter":["red"]`,
		`"PropertyFilters":{"Properties":[{"Property":{"EoCloudCover":{"LowerBound":0,"UpperBound":10}}}]}`, 1))
	if n := filtered["ApproximateResultCount"].(float64); n == 0 || n >= 74 {
		t.Fatalf("cloud filter kept %v of 74", n)
	}

	if status, out := sgeoCall(t, ts, http.MethodPost, "/earth-observation-jobs", `{"Name":"x","ExecutionRoleArn":"`+sgeoTestRole+`",`+query+`,"JobConfig":{"StackConfig":{},"CloudMaskingConfig":{}}}`); status != http.StatusBadRequest {
		t.Fatalf("two job configs: %d %v", status, out)
	}
	if status, _ := sgeoCall(t, ts, http.MethodPost, "/earth-observation-jobs", `{"Name":"x","ExecutionRoleArn":"`+sgeoTestRole+`",`+query+`,"JobConfig":{"StackConfig":{"TargetBands":["nope"]}}}`); status != http.StatusBadRequest {
		t.Fatalf("unknown target band: %d", status)
	}
	job := sgeoOK(t, ts, http.MethodPost, "/earth-observation-jobs", `{"Name":"ndvi","ClientToken":"00000000-0000-0000-0000-000000000001","ExecutionRoleArn":"`+sgeoTestRole+`",`+query+`,"JobConfig":{"BandMathConfig":{"PredefinedIndices":["ndvi"]}},"Tags":{"env":"dev"}}`)
	arn := job["Arn"].(string)
	if job["Status"] != "INITIALIZING" {
		t.Fatalf("job: %v", job)
	}
	// The same client token returns the same job.
	if again := sgeoOK(t, ts, http.MethodPost, "/earth-observation-jobs", `{"Name":"ndvi","ClientToken":"00000000-0000-0000-0000-000000000001","ExecutionRoleArn":"`+sgeoTestRole+`",`+query+`,"JobConfig":{"BandMathConfig":{"PredefinedIndices":["ndvi"]}}}`); again["Arn"] != arn {
		t.Fatalf("client token: %v", again)
	}
	jobPath := sgeoPath("/earth-observation-jobs/", arn)
	tile := "/tile/2/1/1?Arn=" + strings.ReplaceAll(arn, "/", "%2F") + "&ImageAssets=ndvi&Target=OUTPUT"
	if status, out := sgeoCall(t, ts, http.MethodGet, tile, ""); status != http.StatusConflict {
		t.Fatalf("tile before completion: %d %v", status, out)
	}
	done := sgeoAwait(t, ts, jobPath, "Status", "COMPLETED")
	if bands := done["OutputBands"].([]any); len(bands) != 1 || bands[0].(map[string]any)["BandName"] != "ndvi" {
		t.Fatalf("output bands: %v", done)
	}
	if tr := done["InputConfig"].(map[string]any)["RasterDataCollectionQuery"].(map[string]any)["TimeRangeFilter"].(map[string]any); tr["StartTime"] != "2023-01-01T00:00:00Z" {
		t.Fatalf("time range: %v", tr)
	}

	resp := signedRequestWithService(t, http.MethodGet, ts.URL+tile, nil, nil, sgeoSigningName)
	body := mustBody(t, resp)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("tile: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if img, err := png.Decode(bytes.NewReader(body)); err != nil || img.Bounds().Dx() != sgeoTileSize {
		t.Fatalf("tile image: %v", err)
	}
	if status, _ := sgeoCall(t, ts, http.MethodGet, strings.Replace(tile, "/tile/2/1/1", "/tile/2/9/1", 1), ""); status != http.StatusBadRequest {
		t.Fatalf("tile outside the grid: %d", status)
	}

	exp := sgeoOK(t, ts, http.MethodPost, "/export-earth-observation-job", `{"Arn":"`+arn+`","ExecutionRoleArn":"`+sgeoTestRole+`","OutputConfig":{"S3Data":{"S3Uri":"s3://geo-out/eo"}}}`)
	if exp["ExportStatus"] != "IN_PROGRESS" {
		t.Fatalf("export: %v", exp)
	}
	sgeoAwait(t, ts, jobPath, "ExportStatus", "SUCCEEDED")
	id := arn[strings.LastIndex(arn, "/")+1:]
	for _, key := range []string{"eo/" + id + "/manifest.json", "eo/" + id + "/output/ndvi.png"} {
		if _, err := srv.s3.GetObject("geo-out", key); err != nil {
			t.Fatalf("export object %s: %v", key, err)
		}
	}
	sgeoOK(t, ts, http.MethodPost, "/export-earth-observation-job", `{"Arn":"`+arn+`","ExecutionRoleArn":"`+sgeoTestRole+`","OutputConfig":{"S3Data":{"S3Uri":"s3://missing-bucket/eo"}}}`)
	if failed := sgeoAwait(t, ts, jobPath, "ExportStatus", "FAILED"); failed["ExportErrorDetails"] == nil {
		t.Fatalf("failed export: %v", failed)
	}

	// A job over a previous job's output; stop it, then delete both.
	next := sgeoOK(t, ts, http.MethodPost, "/earth-observation-jobs", `{"Name":"stack","ExecutionRoleArn":"`+sgeoTestRole+`","InputConfig":{"PreviousEarthObservationJobArn":"`+arn+`"},"JobConfig":{"TemporalStatisticsConfig":{"Statistics":["MEAN"]}}}`)
	sgeoOK(t, ts, http.MethodPost, "/earth-observation-jobs/stop", `{"Arn":"`+next["Arn"].(string)+`"}`)
	if status, _ := sgeoCall(t, ts, http.MethodPost, "/earth-observation-jobs/stop", `{"Arn":"`+next["Arn"].(string)+`"}`); status != http.StatusConflict {
		t.Fatalf("stop a stopping job: %d", status)
	}
	stopped := sgeoAwait(t, ts, sgeoPath("/earth-observation-jobs/", next["Arn"].(string)), "Status", "STOPPED")
	if bands := stopped["OutputBands"].([]any); bands[0].(map[string]any)["BandName"] != "ndvi_mean" {
		t.Fatalf("temporal bands: %v", bands)
	}
	if list := sgeoOK(t, ts, http.MethodPost, "/list-earth-observation-jobs", `{"StatusEquals":"STOPPED"}`)["EarthObservationJobSummaries"].([]any); len(list) != 1 || list[0].(map[string]any)["OperationType"] != "TEMPORAL_STATISTICS" {
		t.Fatalf("list: %v", list)
	}

	sgeoOK(t, ts, http.MethodPut, sgeoPath("/tags/", arn), `{"Tags":{"team":"geo"}}`)
	sgeoOK(t, ts, http.MethodDelete, sgeoPath("/tags/", arn)+"?tagKeys=env", "")
	if tags := sgeoOK(t, ts, http.MethodGet, sgeoPath("/tags/", arn), "")["Tags"].(map[string]any); len(tags) != 1 || tags["team"] != "geo" {
		t.Fatalf("tags: %v", tags)
	}
	sgeoOK(t, ts, http.MethodDelete, sgeoPath("/earth-observation-jobs/", next["Arn"].(string)), "")
	sgeoOK(t, ts, http.MethodDelete, jobPath, "")
	if status, _ := sgeoCall(t, ts, http.MethodGet, jobPath, ""); status != http.StatusNotFound {
		t.Fatalf("deleted job: %d", status)
	}
}

func TestSageMakerGeospatialVectorEnrichment(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	if _, err := srv.s3.CreateBucket("geo-vec"); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.s3.PutObject("geo-vec", "in/points.csv", "text/csv", []byte("id,lon,lat\n1,-122.33,47.61\n2,-122.30,47.60\n"), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	start := func(uri, x string) string {
		return sgeoOK(t, ts, http.MethodPost, "/vector-enrichment-jobs", `{"Name":"rg","ExecutionRoleArn":"`+sgeoTestRole+`","InputConfig":{"DocumentType":"CSV","DataSourceConfig":{"S3Data":{"S3Uri":"`+uri+`"}}},"JobConfig":{"ReverseGeocodingConfig":{"XAttributeName":"`+x+`","YAttributeName":"lat"}}}`)["Arn"].(string)
	}
	good := start("s3://geo-vec/in/points.csv", "lon")
	bad := start("s3://geo-vec/in/points.csv", "longitude")
	path := sgeoPath("/vector-enrichment-jobs/", good)
	done := sgeoAwait(t, ts, path, "Status", "COMPLETED")
	if done["Type"] != "REVERSE_GEOCODING" {
		t.Fatalf("job: %v", done)
	}
	if failed := sgeoAwait(t, ts, sgeoPath("/vector-enrichment-jobs/", bad), "Status", "FAILED"); !strings.Contains(failed["ErrorDetails"].(map[string]any)["ErrorMessage"].(string), "longitude") {
		t.Fatalf("failed job: %v", failed)
	}
	sgeoOK(t, ts, http.MethodPost, "/export-vector-enrichment-jobs", `{"Arn":"`+good+`","ExecutionRoleArn":"`+sgeoTestRole+`","OutputConfig":{"S3Data":{"S3Uri":"s3://geo-vec/out/"}}}`)
	sgeoAwait(t, ts, path, "ExportStatus", "SUCCEEDED")
	obj, err := srv.s3.GetObject("geo-vec", "out/"+good[strings.LastIndex(good, "/")+1:]+"/output.csv")
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(string(obj.Body)), "\n"); len(lines) != 3 || !strings.Contains(lines[0], "reverse_geo.label") {
		t.Fatalf("output: %q", obj.Body)
	}
	if list := sgeoOK(t, ts, http.MethodPost, "/list-vector-enrichment-jobs", `{"SortBy":"Name","SortOrder":"DESCENDING"}`)["VectorEnrichmentJobSummaries"].([]any); len(list) != 2 {
		t.Fatalf("list: %v", list)
	}
	if status, _ := sgeoCall(t, ts, http.MethodPost, "/vector-enrichment-jobs/stop", `{"Arn":"`+good+`"}`); status != http.StatusConflict {
		t.Fatalf("stop a completed job: %d", status)
	}
	sgeoOK(t, ts, http.MethodDelete, path, "")
}
