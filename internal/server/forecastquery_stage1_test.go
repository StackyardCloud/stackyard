package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestForecastQueryStage1Errors(t *testing.T) {
	ts := newForecastTestServer(t)
	q := func(op, body string) (int, map[string]any) {
		return jsonCall(t, ts, fcqTargetPrefix+op, "forecast", body)
	}
	if status, out := q("QueryForecast", `{"ForecastArn":"arn:aws:forecast:us-east-1:123456789012:forecast/none","Filters":{"item_id":"a"}}`); status != http.StatusBadRequest || out["__type"] != "ResourceNotFoundException" {
		t.Fatalf("missing forecast: %d %v", status, out)
	}
	if status, out := q("QueryForecast", `{"ForecastArn":"arn:aws:forecast:us-east-1:123456789012:forecast/none"}`); status != http.StatusBadRequest || !strings.Contains(fmt.Sprint(out["Message"]), "Filters") {
		t.Fatalf("missing Filters: %d %v", status, out)
	}
	p := fcPipeline(t, ts, "q")
	for _, body := range []string{
		fmt.Sprintf(`{"ForecastArn":%q,"Filters":{"item_id":"a"},"StartDate":"yesterday"}`, p["forecast"]),
		fmt.Sprintf(`{"ForecastArn":%q,"Filters":{"item_id":"a"},"StartDate":"2030-01-02T00:00:00","EndDate":"2030-01-01T00:00:00"}`, p["forecast"]),
	} {
		if status, out := q("QueryForecast", body); status != http.StatusBadRequest || out["__type"] != "InvalidInputException" {
			t.Fatalf("bad window %s: %d %v", body, status, out)
		}
	}
	if status, out := q("QueryForecast", fmt.Sprintf(`{"ForecastArn":%q,"Filters":{"item_id":"a"},"NextToken":"x"}`, p["forecast"])); status != http.StatusBadRequest || out["__type"] != "InvalidNextTokenException" {
		t.Fatalf("next token: %d %v", status, out)
	}
	// A forecast still being created can't be queried.
	early := fcOK(t, ts, "CreateForecast", fmt.Sprintf(`{"ForecastName":"early","PredictorArn":%q}`, p["predictor"]))["ForecastArn"].(string)
	if status, out := q("QueryForecast", fmt.Sprintf(`{"ForecastArn":%q,"Filters":{"item_id":"a"}}`, early)); status != http.StatusBadRequest || out["__type"] != "ResourceInUseException" {
		t.Fatalf("in-progress forecast: %d %v", status, out)
	}
}
