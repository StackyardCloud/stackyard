package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func ebForm(pairs ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		v.Set(pairs[i], pairs[i+1])
	}
	return v
}

// ebExpectError checks a refused request answers with the Query protocol's
// ErrorResponse document carrying a Sender fault and the given code.
func ebExpectError(t *testing.T, ts *httptest.Server, values url.Values, wantStatus int, wantCode string) string {
	t.Helper()
	resp := elasticBeanstalkRequest(t, ts, values)
	assertStatus(t, resp, wantStatus)
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "xml") {
		t.Fatalf("%s: content type %q, want XML", values.Get("Action"), ct)
	}
	body := mustBody(t, resp)
	var doc elasticBeanstalkErrorResponse
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("%s: unmarshal error response %s: %v", values.Get("Action"), body, err)
	}
	if doc.XMLName.Local != "ErrorResponse" || doc.Error.Type != "Sender" || doc.Error.Code != wantCode || doc.Error.Message == "" {
		t.Fatalf("%s: error response %+v, want Sender %s with a message", values.Get("Action"), doc, wantCode)
	}
	return doc.Error.Message
}

func TestElasticBeanstalkStage5ValidationAndErrorShape(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Request validation: a blank name is refused. (Forms without an Action,
	// or with another API version, aren't claimed by Elastic Beanstalk.)
	ebExpectError(t, ts, ebForm("Action", "CreateApplication", "ApplicationName", "  "), http.StatusBadRequest, "InvalidParameterValue")

	// A typed success result inside the modeled envelope.
	resp := elasticBeanstalkRequest(t, ts, ebForm("Action", "CreateApplication", "ApplicationName", "shape-app", "Description", "d"))
	assertStatus(t, resp, http.StatusOK)
	body := mustBody(t, resp)
	var created struct {
		XMLName xml.Name `xml:"CreateApplicationResponse"`
		Xmlns   string   `xml:"xmlns,attr"`
		Result  struct {
			Application struct {
				ApplicationName string
				ApplicationArn  string
				Description     string
			}
		} `xml:"CreateApplicationResult"`
		RequestID string `xml:"ResponseMetadata>RequestId"`
	}
	if err := xml.Unmarshal(body, &created); err != nil {
		t.Fatalf("unmarshal CreateApplication: %v", err)
	}
	app := created.Result.Application
	if created.Xmlns != elasticBeanstalkNamespace || created.RequestID == "" || app.ApplicationName != "shape-app" ||
		app.Description != "d" || !strings.HasSuffix(app.ApplicationArn, ":application/shape-app") {
		t.Fatalf("CreateApplication response %+v", created)
	}

	// The same name again, and operations on names that don't exist, are
	// refused with EB's 400 InvalidParameterValue.
	ebExpectError(t, ts, ebForm("Action", "CreateApplication", "ApplicationName", "shape-app"), http.StatusBadRequest, "InvalidParameterValue")
	ebExpectError(t, ts, ebForm("Action", "UpdateApplication", "ApplicationName", "missing-app", "Description", "x"), http.StatusBadRequest, "InvalidParameterValue")
	ebExpectError(t, ts, ebForm("Action", "DeleteApplication", "ApplicationName", "missing-app"), http.StatusBadRequest, "InvalidParameterValue")

	// The refused duplicate left the original application unchanged.
	resp = elasticBeanstalkRequest(t, ts, ebForm("Action", "DescribeApplications", "ApplicationNames.member.1", "shape-app"))
	assertStatus(t, resp, http.StatusOK)
	var described struct {
		Apps []struct {
			ApplicationName string
			Description     string
		} `xml:"DescribeApplicationsResult>Applications>member"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &described); err != nil {
		t.Fatalf("unmarshal DescribeApplications: %v", err)
	}
	if len(described.Apps) != 1 || described.Apps[0].ApplicationName != "shape-app" || described.Apps[0].Description != "d" {
		t.Fatalf("DescribeApplications %+v", described.Apps)
	}
}
