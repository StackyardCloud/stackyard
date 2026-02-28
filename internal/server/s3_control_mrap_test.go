package server

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestS3ControlMultiRegionAccessPointsStage4(t *testing.T) {
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

	resp := signedRequest(t, http.MethodPut, ts.URL+"/mrap-bucket-1", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, http.MethodPut, ts.URL+"/mrap-bucket-2", nil, nil)
	assertStatus(t, resp, http.StatusOK)

	badOrderBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateMultiRegionAccessPointRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Details>` +
		`<Name>mrap-demo</Name>` +
		`<Regions>` +
		`<Region><Bucket>mrap-bucket-1</Bucket><BucketAccountId>` + accountID + `</BucketAccountId><Region>us-east-1</Region></Region>` +
		`</Regions>` +
		`</Details>` +
		`<ClientToken>token-1</ClientToken>` +
		`</CreateMultiRegionAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/async-requests/mrap/create", []byte(badOrderBody), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid order, got %d", resp.StatusCode)
	}

	badDetailsOrder := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateMultiRegionAccessPointRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<ClientToken>token-1</ClientToken>` +
		`<Details>` +
		`<Regions>` +
		`<Region><Bucket>mrap-bucket-1</Bucket><BucketAccountId>` + accountID + `</BucketAccountId><Region>us-east-1</Region></Region>` +
		`</Regions>` +
		`<Name>mrap-demo</Name>` +
		`</Details>` +
		`</CreateMultiRegionAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/async-requests/mrap/create", []byte(badDetailsOrder), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid details order, got %d", resp.StatusCode)
	}

	badPabOrder := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateMultiRegionAccessPointRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<ClientToken>token-1</ClientToken>` +
		`<Details>` +
		`<Name>mrap-demo</Name>` +
		`<PublicAccessBlock>` +
		`<BlockPublicAcls>false</BlockPublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<RestrictPublicBuckets>false</RestrictPublicBuckets>` +
		`</PublicAccessBlock>` +
		`<Regions>` +
		`<Region><Bucket>mrap-bucket-1</Bucket><BucketAccountId>` + accountID + `</BucketAccountId><Region>us-east-1</Region></Region>` +
		`</Regions>` +
		`</Details>` +
		`</CreateMultiRegionAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/async-requests/mrap/create", []byte(badPabOrder), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid PAB order, got %d", resp.StatusCode)
	}

	badPabBool := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateMultiRegionAccessPointRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<ClientToken>token-1</ClientToken>` +
		`<Details>` +
		`<Name>mrap-demo</Name>` +
		`<PublicAccessBlock>` +
		`<BlockPublicAcls>yes</BlockPublicAcls>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`<RestrictPublicBuckets>false</RestrictPublicBuckets>` +
		`</PublicAccessBlock>` +
		`<Regions>` +
		`<Region><Bucket>mrap-bucket-1</Bucket><BucketAccountId>` + accountID + `</BucketAccountId><Region>us-east-1</Region></Region>` +
		`</Regions>` +
		`</Details>` +
		`</CreateMultiRegionAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/async-requests/mrap/create", []byte(badPabBool), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid PAB bool, got %d", resp.StatusCode)
	}

	createBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateMultiRegionAccessPointRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<ClientToken>token-1</ClientToken>` +
		`<Details>` +
		`<Name>mrap-demo</Name>` +
		`<PublicAccessBlock>` +
		`<BlockPublicAcls>false</BlockPublicAcls>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`<RestrictPublicBuckets>false</RestrictPublicBuckets>` +
		`</PublicAccessBlock>` +
		`<Regions>` +
		`<Region><Bucket>mrap-bucket-1</Bucket><BucketAccountId>` + accountID + `</BucketAccountId><Region>us-east-1</Region></Region>` +
		`<Region><Bucket>mrap-bucket-2</Bucket><BucketAccountId>` + accountID + `</BucketAccountId><Region>us-west-2</Region></Region>` +
		`</Regions>` +
		`</Details>` +
		`</CreateMultiRegionAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/async-requests/mrap/create", []byte(createBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3ControlCreateMultiRegionAccessPointResult
	if err := xml.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create mrap response: %v", err)
	}
	if createResp.RequestTokenARN == "" {
		t.Fatalf("expected request token ARN")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/async-requests/mrap/"+createResp.RequestTokenARN, nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var describeResp s3ControlDescribeMultiRegionAccessPointOperationResult
	if err := xml.Unmarshal(mustBody(t, resp), &describeResp); err != nil {
		t.Fatalf("parse describe operation response: %v", err)
	}
	if describeResp.AsyncOperation.RequestStatus != "SUCCEEDED" {
		t.Fatalf("expected operation succeeded, got %q", describeResp.AsyncOperation.RequestStatus)
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/mrap/instances", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listResp s3ControlListMultiRegionAccessPointsResult
	if err := xml.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("parse list mrap response: %v", err)
	}
	if len(listResp.AccessPoints) != 1 {
		t.Fatalf("expected 1 mrap, got %d", len(listResp.AccessPoints))
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/mrap/instances/mrap-demo", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getResp s3ControlGetMultiRegionAccessPointResult
	if err := xml.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse get mrap response: %v", err)
	}
	if getResp.AccessPoint.Name != "mrap-demo" {
		t.Fatalf("unexpected mrap name: %q", getResp.AccessPoint.Name)
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"*"}]}`

	badPolicyOrder := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PutMultiRegionAccessPointPolicyRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Details><Name>mrap-demo</Name><Policy>{}</Policy></Details>` +
		`<ClientToken>token-2</ClientToken>` +
		`</PutMultiRegionAccessPointPolicyRequest>`
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/async-requests/mrap/put-policy", []byte(badPolicyOrder), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for policy order, got %d", resp.StatusCode)
	}

	var policyBuf bytes.Buffer
	policyBuf.WriteString(`<PutMultiRegionAccessPointPolicyRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">`)
	policyBuf.WriteString(`<ClientToken>token-2</ClientToken><Details><Name>mrap-demo</Name><Policy>`)
	if err := xml.EscapeText(&policyBuf, []byte(policyJSON)); err != nil {
		t.Fatalf("escape policy: %v", err)
	}
	policyBuf.WriteString(`</Policy></Details></PutMultiRegionAccessPointPolicyRequest>`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/async-requests/mrap/put-policy", policyBuf.Bytes(), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var putPolicyResp s3ControlPutMultiRegionAccessPointPolicyResult
	if err := xml.Unmarshal(mustBody(t, resp), &putPolicyResp); err != nil {
		t.Fatalf("parse put policy response: %v", err)
	}
	if putPolicyResp.RequestTokenARN == "" {
		t.Fatalf("expected request token for policy")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/mrap/instances/mrap-demo/policy", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getPolicyResp s3ControlGetMultiRegionAccessPointPolicyResult
	if err := xml.Unmarshal(mustBody(t, resp), &getPolicyResp); err != nil {
		t.Fatalf("parse get policy response: %v", err)
	}
	if getPolicyResp.Policy.Established == nil || strings.TrimSpace(getPolicyResp.Policy.Established.Policy) != policyJSON {
		t.Fatalf("unexpected policy body")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/mrap/instances/mrap-demo/policyStatus", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var statusResp s3ControlGetMultiRegionAccessPointPolicyStatusResult
	if err := xml.Unmarshal(mustBody(t, resp), &statusResp); err != nil {
		t.Fatalf("parse policy status response: %v", err)
	}
	if !statusResp.Established.IsPublic {
		t.Fatalf("expected policy status public")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/mrap/instances/mrap-demo/routes", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var routesResp s3ControlGetMultiRegionAccessPointRoutesResult
	if err := xml.Unmarshal(mustBody(t, resp), &routesResp); err != nil {
		t.Fatalf("parse get routes response: %v", err)
	}
	if len(routesResp.Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routesResp.Routes))
	}

	invalidRoutes := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<SubmitMultiRegionAccessPointRoutesRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<RouteUpdates>` +
		`<Route><Bucket>mrap-bucket-1</Bucket><Region>us-east-1</Region><TrafficDialPercentage>0</TrafficDialPercentage></Route>` +
		`<Route><Bucket>mrap-bucket-2</Bucket><Region>us-west-2</Region><TrafficDialPercentage>0</TrafficDialPercentage></Route>` +
		`</RouteUpdates></SubmitMultiRegionAccessPointRoutesRequest>`
	resp = signedRequestWithService(t, http.MethodPatch, ts.URL+"/v20180820/mrap/instances/mrap-demo/routes", []byte(invalidRoutes), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for all-zero routes, got %d", resp.StatusCode)
	}

	badRouteOrder := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<SubmitMultiRegionAccessPointRoutesRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<RouteUpdates>` +
		`<Route><Region>us-east-1</Region><Bucket>mrap-bucket-1</Bucket><TrafficDialPercentage>100</TrafficDialPercentage></Route>` +
		`</RouteUpdates></SubmitMultiRegionAccessPointRoutesRequest>`
	resp = signedRequestWithService(t, http.MethodPatch, ts.URL+"/v20180820/mrap/instances/mrap-demo/routes", []byte(badRouteOrder), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for route order, got %d", resp.StatusCode)
	}

	updateRoutes := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<SubmitMultiRegionAccessPointRoutesRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<RouteUpdates>` +
		`<Route><Bucket>mrap-bucket-1</Bucket><Region>us-east-1</Region><TrafficDialPercentage>0</TrafficDialPercentage></Route>` +
		`<Route><Bucket>mrap-bucket-2</Bucket><Region>us-west-2</Region><TrafficDialPercentage>100</TrafficDialPercentage></Route>` +
		`</RouteUpdates></SubmitMultiRegionAccessPointRoutesRequest>`
	resp = signedRequestWithService(t, http.MethodPatch, ts.URL+"/v20180820/mrap/instances/mrap-demo/routes", []byte(updateRoutes), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
}
