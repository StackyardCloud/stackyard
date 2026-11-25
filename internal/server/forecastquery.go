package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Forecast Query: awsJson1_1, X-Amz-Target AmazonForecastRuntime.*,
// signed forecast. Queries read forecasts and what-if forecasts from the
// Forecast store.

const fcqTargetPrefix = "AmazonForecastRuntime."

func (s *Server) handleForecastQueryJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), fcqTargetPrefix)
	if !ok {
		return false
	}
	model := awsmodels.ForecastQueryModel()
	in, ferr := fcDecode(s, w, r, "forecast", model, op)
	if ferr != nil {
		respondForecastError(w, ferr)
		return true
	}
	if in == nil {
		return true
	}
	out, ferr := s.forecast.runQuery(op, in)
	if ferr != nil {
		respondForecastError(w, ferr)
		return true
	}
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func (st *forecastStore) runQuery(op string, in map[string]any) (any, *fcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var out map[string]any
	var err *fcErr
	switch op {
	case "QueryForecast":
		out, err = st.query(fcStr(in, "ForecastArn"), false, in)
	case "QueryWhatIfForecast":
		out, err = st.query(fcStr(in, "WhatIfForecastArn"), true, in)
	default:
		return nil, fcError("UnknownOperationException", "unknown operation %s", op)
	}
	if err != nil {
		return nil, err
	}
	model := awsmodels.ForecastQueryModel()
	return model.Conform(model.Operations[op].Output, out), nil
}

// fcqWindow reads the optional StartDate/EndDate window.
func fcqWindow(in map[string]any) (time.Time, time.Time, *fcErr) {
	var out [2]time.Time
	for i, key := range []string{"StartDate", "EndDate"} {
		if s := fcStr(in, key); s != "" {
			t, err := time.Parse(fcqTimeLayout, s)
			if err != nil {
				return time.Time{}, time.Time{}, fcInvalid("invalid %s: use the form yyyy-MM-ddTHH:mm:ss", key)
			}
			out[i] = t
		}
	}
	if !out[0].IsZero() && !out[1].IsZero() && out[1].Before(out[0]) {
		return time.Time{}, time.Time{}, fcInvalid("invalid window: EndDate is before StartDate")
	}
	return out[0], out[1], nil
}
