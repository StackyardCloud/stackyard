package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECRStage11ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const repositoryName = "stage11-raw"
	resp := ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"`+repositoryName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	subjectPayload, err := json.Marshal(map[string]any{
		"repositoryName": repositoryName,
		"imageManifest":  `{"schemaVersion":2}`,
		"imageTag":       "subject",
	})
	if err != nil {
		t.Fatalf("marshal subject put image payload: %v", err)
	}
	resp = ecrRequest(t, ts, "PutImage", subjectPayload)
	assertStatus(t, resp, http.StatusOK)

	var subjectOut struct {
		Image struct {
			ImageID struct {
				ImageDigest string `json:"imageDigest"`
			} `json:"imageId"`
		} `json:"image"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &subjectOut); err != nil {
		t.Fatalf("unmarshal subject put image output: %v", err)
	}
	if subjectOut.Image.ImageID.ImageDigest == "" {
		t.Fatalf("expected subject image digest")
	}

	referrerManifest, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.manifest.v1+json",
		"artifactType":  "application/vnd.example.signature",
		"subject": map[string]any{
			"digest": subjectOut.Image.ImageID.ImageDigest,
		},
	})
	if err != nil {
		t.Fatalf("marshal referrer manifest: %v", err)
	}
	referrerPayload, err := json.Marshal(map[string]any{
		"repositoryName": repositoryName,
		"imageManifest":  string(referrerManifest),
		"imageTag":       "signature",
	})
	if err != nil {
		t.Fatalf("marshal referrer put image payload: %v", err)
	}
	resp = ecrRequest(t, ts, "PutImage", referrerPayload)
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		name string
		body []byte
	}{
		{name: "RegisterPullTimeUpdateExclusion", body: []byte(`{"principalArn":"arn:aws:iam::123456789012:role/example"}`)},
		{name: "ListPullTimeUpdateExclusions", body: []byte(`{}`)},
		{name: "ListImageReferrers", body: []byte(`{"repositoryName":"` + repositoryName + `","subjectId":{"imageDigest":"` + subjectOut.Image.ImageID.ImageDigest + `"},"filter":{"artifactStatus":"ANY"}}`)},
		{name: "UpdateImageStorageClass", body: []byte(`{"repositoryName":"` + repositoryName + `","imageId":{"imageTag":"subject"},"targetStorageClass":"ARCHIVE"}`)},
		{name: "DeregisterPullTimeUpdateExclusion", body: []byte(`{"principalArn":"arn:aws:iam::123456789012:role/example"}`)},
	}

	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}
