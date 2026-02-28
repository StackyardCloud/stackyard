package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSimSpaceWeaverStage12SimulationAndAppLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	simulation := "stage-simspaceweaver-sim"
	domain := "stage-domain"
	app := "stage-app"

	resp := simSpaceWeaverRequest(t, ts, http.MethodPost, "/startsimulation", []byte(`{
		"Name":"`+simulation+`",
		"RoleArn":"arn:aws:iam::123456789012:role/stackyard-simspaceweaver",
		"SchemaS3Location":{"BucketName":"stackyard-simspaceweaver","ObjectKey":"schemas/simulation-schema.zip"}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = simSpaceWeaverRequest(t, ts, http.MethodGet, "/describesimulation?simulation="+url.QueryEscape(simulation), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, simulation) {
		t.Fatalf("expected DescribeSimulation to include simulation name, got %q", body)
	}

	resp = simSpaceWeaverRequest(t, ts, http.MethodPost, "/createsnapshot", []byte(`{
		"Simulation":"`+simulation+`",
		"Destination":{"BucketName":"stackyard-simspaceweaver","ObjectKeyPrefix":"snapshots"}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = simSpaceWeaverRequest(t, ts, http.MethodGet, "/listsimulations?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "simulations") {
		t.Fatalf("expected ListSimulations to include simulations, got %q", body)
	}

	resp = simSpaceWeaverRequest(t, ts, http.MethodPost, "/startapp", []byte(`{
		"Simulation":"`+simulation+`",
		"Domain":"`+domain+`",
		"Name":"`+app+`"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = simSpaceWeaverRequest(t, ts, http.MethodGet, "/describeapp?app="+url.QueryEscape(app)+"&domain="+url.QueryEscape(domain)+"&simulation="+url.QueryEscape(simulation), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, app) {
		t.Fatalf("expected DescribeApp to include app name, got %q", body)
	}

	resp = simSpaceWeaverRequest(t, ts, http.MethodGet, "/listapps?domain="+url.QueryEscape(domain)+"&simulation="+url.QueryEscape(simulation), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "apps") {
		t.Fatalf("expected ListApps to include apps, got %q", body)
	}
}

func TestSimSpaceWeaverStage34ClockTaggingAndTeardown(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	simulation := "stage-simspaceweaver-sim"
	domain := "stage-domain"
	app := "stage-app"
	resourceARN := url.PathEscape("arn:aws:simspaceweaver:us-east-1:123456789012:simulation/" + simulation)

	resp := simSpaceWeaverRequest(t, ts, http.MethodPost, "/startclock", []byte(`{"Simulation":"`+simulation+`","Domain":"`+domain+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = simSpaceWeaverRequest(t, ts, http.MethodPost, "/stopclock", []byte(`{"Simulation":"`+simulation+`","Domain":"`+domain+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = simSpaceWeaverRequest(t, ts, http.MethodPost, "/tags/"+resourceARN, []byte(`{"tags":{"env":"stage","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = simSpaceWeaverRequest(t, ts, http.MethodGet, "/tags/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = simSpaceWeaverRequest(t, ts, http.MethodDelete, "/tags/"+resourceARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = simSpaceWeaverRequest(t, ts, http.MethodPost, "/stopapp", []byte(`{"Simulation":"`+simulation+`","Domain":"`+domain+`","App":"`+app+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = simSpaceWeaverRequest(t, ts, http.MethodDelete, "/deleteapp?app="+url.QueryEscape(app)+"&domain="+url.QueryEscape(domain)+"&simulation="+url.QueryEscape(simulation), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = simSpaceWeaverRequest(t, ts, http.MethodPost, "/stopsimulation", []byte(`{"Simulation":"`+simulation+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = simSpaceWeaverRequest(t, ts, http.MethodDelete, "/deletesimulation?simulation="+url.QueryEscape(simulation), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestSimSpaceWeaverStage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	simulation := "idempotent-simspaceweaver-sim"

	resp := simSpaceWeaverRequest(t, ts, http.MethodDelete, "/deletesimulation?simulation="+url.QueryEscape(simulation), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = simSpaceWeaverRequest(t, ts, http.MethodDelete, "/deletesimulation?simulation="+url.QueryEscape(simulation), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = simSpaceWeaverRequest(t, ts, http.MethodPost, "/simspaceweaver/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/startsimulation",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"simspaceweaver",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
