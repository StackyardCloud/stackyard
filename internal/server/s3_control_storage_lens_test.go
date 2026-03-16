package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3ControlStorageLensStage6(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	accountID := "123456789012"
	headers := map[string]string{
		"x-amz-account-id": accountID,
		"Content-Type":     "application/xml",
	}

	putConfig := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PutStorageLensConfigurationRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<StorageLensConfiguration><Id>cfg-1</Id><AccountLevel></AccountLevel></StorageLensConfiguration>` +
		`</PutStorageLensConfigurationRequest>`
	resp := signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/storagelens/cfg-1", []byte(putConfig), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/storagelens/cfg-1", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getResp s3ControlGetStorageLensConfigurationResult
	if err := xml.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse get storage lens response: %v", err)
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/storagelens", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listResp s3ControlListStorageLensConfigurationsResult
	if err := xml.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("parse list storage lens response: %v", err)
	}
	if len(listResp.Configurations) != 1 {
		t.Fatalf("expected 1 storage lens config, got %d", len(listResp.Configurations))
	}

	tagBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PutStorageLensConfigurationTaggingRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Tagging><Tag><Key>env</Key><Value>dev</Value></Tag></Tagging>` +
		`</PutStorageLensConfigurationTaggingRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/storagelens/cfg-1/tagging", []byte(tagBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/storagelens/cfg-1/tagging", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var tagResp s3ControlGetStorageLensTaggingResult
	if err := xml.Unmarshal(mustBody(t, resp), &tagResp); err != nil {
		t.Fatalf("parse get tagging response: %v", err)
	}
	if len(tagResp.Tags.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tagResp.Tags.Tags))
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/storagelens/cfg-1/tagging", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/storagelens/cfg-1", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	groupBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateStorageLensGroupRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<StorageLensGroup><Name>group-1</Name></StorageLensGroup>` +
		`</CreateStorageLensGroupRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/storage-lens-groups", []byte(groupBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/storage-lens-groups", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var groupListResp s3ControlListStorageLensGroupsResult
	if err := xml.Unmarshal(mustBody(t, resp), &groupListResp); err != nil {
		t.Fatalf("parse list groups response: %v", err)
	}
	if len(groupListResp.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groupListResp.Groups))
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/storage-lens-groups/group-1", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getGroupResp s3ControlGetStorageLensGroupResult
	if err := xml.Unmarshal(mustBody(t, resp), &getGroupResp); err != nil {
		t.Fatalf("parse get group response: %v", err)
	}

	updateGroup := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<UpdateStorageLensGroupRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<StorageLensGroup><Name>group-1</Name><Filter></Filter></StorageLensGroup>` +
		`</UpdateStorageLensGroupRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/storage-lens-groups/group-1", []byte(updateGroup), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/storage-lens-groups/group-1", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
}
