package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3ControlAccessGrantsStage7(t *testing.T) {
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

	createInstance := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateAccessGrantsInstanceRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<IdentityCenterArn>arn:aws:sso:::instance/ssoins-123</IdentityCenterArn>` +
		`<Tags><Tag><Key>env</Key><Value>test</Value></Tag></Tags>` +
		`</CreateAccessGrantsInstanceRequest>`
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/accessgrantsinstance", []byte(createInstance), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var createInstanceResp s3ControlAccessGrantsInstanceResult
	if err := xml.Unmarshal(mustBody(t, resp), &createInstanceResp); err != nil {
		t.Fatalf("parse create access grants instance response: %v", err)
	}
	if createInstanceResp.AccessGrantsInstanceId == "" {
		t.Fatalf("expected access grants instance id")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getInstanceResp s3ControlAccessGrantsInstanceResult
	if err := xml.Unmarshal(mustBody(t, resp), &getInstanceResp); err != nil {
		t.Fatalf("parse get access grants instance response: %v", err)
	}
	if getInstanceResp.AccessGrantsInstanceId != createInstanceResp.AccessGrantsInstanceId {
		t.Fatalf("unexpected instance id")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstances", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listInstancesResp s3ControlListAccessGrantsInstancesResult
	if err := xml.Unmarshal(mustBody(t, resp), &listInstancesResp); err != nil {
		t.Fatalf("parse list access grants instances response: %v", err)
	}
	if len(listInstancesResp.AccessGrantsInstances) != 1 {
		t.Fatalf("expected 1 access grants instance, got %d", len(listInstancesResp.AccessGrantsInstances))
	}

	policyBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PutAccessGrantsInstanceResourcePolicyRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Policy>{"Version":"2012-10-17","Statement":[]}</Policy>` +
		`<Organization>o-123456</Organization>` +
		`</PutAccessGrantsInstanceResourcePolicyRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accessgrantsinstance/resourcepolicy", []byte(policyBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance/resourcepolicy", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getPolicyResp s3ControlGetAccessGrantsInstanceResourcePolicyResult
	if err := xml.Unmarshal(mustBody(t, resp), &getPolicyResp); err != nil {
		t.Fatalf("parse get access grants resource policy response: %v", err)
	}
	if getPolicyResp.Policy == "" {
		t.Fatalf("expected policy in resource policy response")
	}

	assocBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<AssociateAccessGrantsIdentityCenterRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<IdentityCenterArn>arn:aws:sso:::instance/ssoins-123</IdentityCenterArn>` +
		`</AssociateAccessGrantsIdentityCenterRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/accessgrantsinstance/identitycenter", []byte(assocBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	badLocationOrder := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateAccessGrantsLocationRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<IAMRoleArn>arn:aws:iam::123456789012:role/Test</IAMRoleArn>` +
		`<LocationScope>s3://example-bucket/prefix</LocationScope>` +
		`</CreateAccessGrantsLocationRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/accessgrantsinstance/location", []byte(badLocationOrder), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid location order, got %d", resp.StatusCode)
	}

	createLocation := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateAccessGrantsLocationRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<LocationScope>s3://example-bucket/prefix</LocationScope>` +
		`<IAMRoleArn>arn:aws:iam::123456789012:role/Test</IAMRoleArn>` +
		`</CreateAccessGrantsLocationRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/accessgrantsinstance/location", []byte(createLocation), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var createLocationResp s3ControlAccessGrantsLocationResult
	if err := xml.Unmarshal(mustBody(t, resp), &createLocationResp); err != nil {
		t.Fatalf("parse create access grants location response: %v", err)
	}
	if createLocationResp.AccessGrantsLocationId == "" {
		t.Fatalf("expected access grants location id")
	}
	locationID := createLocationResp.AccessGrantsLocationId

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance/location/"+locationID, nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	updateLocation := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<UpdateAccessGrantsLocationRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<IAMRoleArn>arn:aws:iam::123456789012:role/Updated</IAMRoleArn>` +
		`</UpdateAccessGrantsLocationRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accessgrantsinstance/location/"+locationID, []byte(updateLocation), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance/locations", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listLocationsResp s3ControlListAccessGrantsLocationsResult
	if err := xml.Unmarshal(mustBody(t, resp), &listLocationsResp); err != nil {
		t.Fatalf("parse list access grants locations response: %v", err)
	}
	if len(listLocationsResp.AccessGrantsLocations) != 1 {
		t.Fatalf("expected 1 access grants location, got %d", len(listLocationsResp.AccessGrantsLocations))
	}

	createGrant := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateAccessGrantRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<AccessGrantsLocationId>` + locationID + `</AccessGrantsLocationId>` +
		`<AccessGrantsLocationConfiguration><S3SubPrefix>docs/</S3SubPrefix></AccessGrantsLocationConfiguration>` +
		`<Grantee><GranteeType>IAM</GranteeType><GranteeIdentifier>arn:aws:iam::123456789012:user/Alice</GranteeIdentifier></Grantee>` +
		`<Permission>READ</Permission>` +
		`<S3PrefixType>Object</S3PrefixType>` +
		`</CreateAccessGrantRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/accessgrantsinstance/grant", []byte(createGrant), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var createGrantResp s3ControlAccessGrantResult
	if err := xml.Unmarshal(mustBody(t, resp), &createGrantResp); err != nil {
		t.Fatalf("parse create access grant response: %v", err)
	}
	if createGrantResp.AccessGrantId == "" {
		t.Fatalf("expected access grant id")
	}
	grantID := createGrantResp.AccessGrantId

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance/grant/"+grantID, nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance/grants", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listGrantsResp s3ControlListAccessGrantsResult
	if err := xml.Unmarshal(mustBody(t, resp), &listGrantsResp); err != nil {
		t.Fatalf("parse list access grants response: %v", err)
	}
	if len(listGrantsResp.AccessGrants) != 1 {
		t.Fatalf("expected 1 access grant, got %d", len(listGrantsResp.AccessGrants))
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/accessgrantsinstance/grant/"+grantID, nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance/grant/"+grantID, nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for deleted access grant, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/accessgrantsinstance/location/"+locationID, nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance/location/"+locationID, nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for deleted access grants location, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/accessgrantsinstance/resourcepolicy", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance/resourcepolicy", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after deleting resource policy, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/accessgrantsinstance", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accessgrantsinstance", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after deleting access grants instance, got %d", resp.StatusCode)
	}
}
