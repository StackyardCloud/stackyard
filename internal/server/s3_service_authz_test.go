package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// svcAuthzPolicy denies (or allows) every S3 action on bucket to principal.
func svcAuthzPolicy(effect, principal, bucket string) string {
	return fmt.Sprintf(`{"Statement":[{"Effect":%q,"Principal":%s,"Action":"s3:*","Resource":["arn:aws:s3:::%s","arn:aws:s3:::%s/*"]}]}`,
		effect, principal, bucket, bucket)
}

// svcAuthzCases are the three bucket states every service is checked under.
func svcAuthzCases(service string) []struct{ name, principal string } {
	return []struct{ name, principal string }{
		{"no policy", ""},
		{"caller denied", `{"AWS":"` + testAccessKey + `"}`},
		{"service denied", `{"Service":"` + service + `"}`},
	}
}

func svcAuthzBucket(t *testing.T, srv *Server, bucket, principal string, objects ...string) {
	t.Helper()
	if _, err := srv.s3.CreateBucket(bucket); err != nil {
		t.Fatal(err)
	}
	for _, k := range objects {
		if _, err := srv.s3.PutObject(bucket, k, "text/plain", []byte("Hello world\nSecond line\n"), nil, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if principal != "" {
		if err := srv.s3.SetBucketPolicy(bucket, svcAuthzPolicy("Deny", principal, bucket)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestServicePrincipalMatching(t *testing.T) {
	for _, tc := range []struct {
		name      string
		principal any
		user, svc bool
	}{
		{"star matches both", "*", true, true},
		{"service string", map[string]any{"Service": "textract.amazonaws.com"}, false, true},
		{"service list", map[string]any{"Service": []any{"polly.amazonaws.com", "textract.amazonaws.com"}}, false, true},
		{"other service", map[string]any{"Service": "polly.amazonaws.com"}, false, false},
		{"user principal", map[string]any{"AWS": "textract.amazonaws.com"}, true, false},
		{"aws star is users only", map[string]any{"AWS": "*"}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The user's credential is spelled like the service on purpose: a
			// user must still never match a Service principal.
			if got := principalMatches(tc.principal, "textract.amazonaws.com"); got != tc.user {
				t.Fatalf("user match = %v, want %v", got, tc.user)
			}
			if got := servicePrincipalMatches(tc.principal, "textract.amazonaws.com"); got != tc.svc {
				t.Fatalf("service match = %v, want %v", got, tc.svc)
			}
		})
	}
}

func TestAuthorizeServiceS3(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	const svc = "textract.amazonaws.com"
	signed := s3Caller{requester: testAccessKey, signed: true}
	anon := s3Caller{requester: "anonymous"}
	put := func(bucket, acl, policy string) {
		if _, err := srv.s3.CreateBucket(bucket); err != nil {
			t.Fatal(err)
		}
		if _, err := srv.s3.PutObject(bucket, "k", "text/plain", []byte("x"), nil, acl, ""); err != nil {
			t.Fatal(err)
		}
		if policy != "" {
			if err := srv.s3.SetBucketPolicy(bucket, policy); err != nil {
				t.Fatal(err)
			}
		}
	}
	put("plain", "", "")
	put("public", "public-read", "")
	put("deny-svc", "", svcAuthzPolicy("Deny", `{"Service":"`+svc+`"}`, "deny-svc"))
	put("deny-user", "", svcAuthzPolicy("Deny", `{"AWS":"`+testAccessKey+`"}`, "deny-user"))
	put("allow-svc", "", svcAuthzPolicy("Allow", `{"Service":"`+svc+`"}`, "allow-svc"))
	put("deny-all", "", svcAuthzPolicy("Deny", `"*"`, "deny-all"))
	// Deny wins over Allow, whichever principal each names.
	put("mixed", "", `{"Statement":[`+
		`{"Effect":"Allow","Principal":{"AWS":"`+testAccessKey+`"},"Action":"s3:*","Resource":"arn:aws:s3:::mixed/*"},`+
		`{"Effect":"Deny","Principal":{"Service":"`+svc+`"},"Action":"s3:GetObject","Resource":"arn:aws:s3:::mixed/*"}]}`)

	for _, tc := range []struct {
		name, bucket, service string
		caller                s3Caller
		want                  bool
	}{
		{"signed, no policy", "plain", svc, signed, true},
		{"anonymous, private object", "plain", svc, anon, false},
		{"anonymous, public-read object", "public", svc, anon, true},
		{"service denied", "deny-svc", svc, signed, false},
		{"other service not denied", "deny-svc", "polly.amazonaws.com", signed, true},
		{"caller denied", "deny-user", svc, signed, false},
		{"service allowed admits anonymous caller", "allow-svc", svc, anon, true},
		{"star denies both", "deny-all", svc, signed, false},
		{"deny wins over allow", "mixed", svc, signed, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := srv.authorizeServiceS3As(tc.caller, tc.service, tc.bucket, "k", "s3:GetObject"); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	// "bucket/*" covers nested keys, as IAM's wildcard crosses "/".
	if srv.authorizeServiceS3As(signed, svc, "deny-svc", "out/job/1", "s3:PutObject") {
		t.Fatal("nested key escaped a bucket/* Deny")
	}
	// The caller rides through a service's input map; JSON can't forge it.
	in := map[string]any{s3CallerKey: map[string]any{"requester": "x", "signed": true}}
	if c := s3CallerIn(in); c.signed {
		t.Fatalf("forged caller accepted: %+v", c)
	}
}

// Sync read: Textract DetectDocumentText on an S3 document.
func TestTextractServiceS3Authz(t *testing.T) {
	for _, tc := range svcAuthzCases(txServicePrincipal) {
		t.Run(tc.name, func(t *testing.T) {
			srv, ts := newTextractTestServer(t)
			svcAuthzBucket(t, srv, "tx-authz", tc.principal, "doc.txt")
			status, out := txCall(t, ts, "DetectDocumentText", `{"Document":{"S3Object":{"Bucket":"tx-authz","Name":"doc.txt"}}}`)
			if tc.principal == "" {
				if status != http.StatusOK {
					t.Fatalf("want 200, got %d %v", status, out)
				}
				return
			}
			if status != http.StatusBadRequest || out["__type"] != "AccessDeniedException" {
				t.Fatalf("want AccessDeniedException, got %d %v", status, out)
			}
		})
	}
}

// Async write: Textract StartDocumentTextDetection's OutputConfig, written
// when the job completes. The input lives in a separate, open bucket.
func TestTextractJobOutputServiceS3Authz(t *testing.T) {
	for _, tc := range svcAuthzCases(txServicePrincipal) {
		t.Run(tc.name, func(t *testing.T) {
			srv, ts := newTextractTestServer(t)
			svcAuthzBucket(t, srv, "tx-in", "", "doc.txt")
			svcAuthzBucket(t, srv, "tx-out", tc.principal)
			job := txOK(t, ts, "StartDocumentTextDetection", `{"DocumentLocation":{"S3Object":{"Bucket":"tx-in","Name":"doc.txt"}},"OutputConfig":{"S3Bucket":"tx-out","S3Prefix":"out"}}`)["JobId"].(string)
			txOK(t, ts, "GetDocumentTextDetection", `{"JobId":"`+job+`"}`) // IN_PROGRESS
			got := txOK(t, ts, "GetDocumentTextDetection", `{"JobId":"`+job+`"}`)
			objs, _ := srv.s3.ListObjects("tx-out", "")
			if tc.principal == "" {
				if got["JobStatus"] != "SUCCEEDED" || len(objs) != 1 {
					t.Fatalf("want SUCCEEDED with output, got %v (%d objects)", got["JobStatus"], len(objs))
				}
				return
			}
			if got["JobStatus"] != "FAILED" || got["StatusMessage"] == nil || len(objs) != 0 {
				t.Fatalf("want FAILED without output, got %v (%d objects)", got, len(objs))
			}
			// The failure sticks on later reads.
			if again := txOK(t, ts, "GetDocumentTextDetection", `{"JobId":"`+job+`"}`); again["JobStatus"] != "FAILED" {
				t.Fatalf("later read: %v", again)
			}
		})
	}
}

// Sync write: Polly StartSpeechSynthesisTask writes the audio to S3.
func TestPollyServiceS3Authz(t *testing.T) {
	for _, tc := range svcAuthzCases(pollyServicePrincipal) {
		t.Run(tc.name, func(t *testing.T) {
			srv, ts := newPollyTestServer(t)
			svcAuthzBucket(t, srv, "polly-authz", tc.principal)
			status, _, data := pollyRaw(t, ts, http.MethodPost, "/v1/synthesisTasks", `{"OutputFormat":"mp3","OutputS3BucketName":"polly-authz","Text":"hello","VoiceId":"Joanna"}`, nil)
			objs, _ := srv.s3.ListObjects("polly-authz", "")
			if tc.principal == "" {
				if status != http.StatusOK || len(objs) != 1 {
					t.Fatalf("want 200 with audio, got %d %s (%d objects)", status, data, len(objs))
				}
				return
			}
			if status != http.StatusForbidden || len(objs) != 0 {
				t.Fatalf("want 403 without audio, got %d %s (%d objects)", status, data, len(objs))
			}
		})
	}
}

// Async copy: a DataSync execution between two buckets, the destination of
// which carries the policy.
func TestDataSyncServiceS3Authz(t *testing.T) {
	for _, tc := range svcAuthzCases("datasync.amazonaws.com") {
		t.Run(tc.name, func(t *testing.T) {
			srv, ts := newDataSyncTestServer(t)
			src, dst := dsyS3Pair(t, srv, ts, "a.txt")
			if tc.principal != "" {
				if err := srv.s3.SetBucketPolicy("dsy-dst", svcAuthzPolicy("Deny", tc.principal, "dsy-dst")); err != nil {
					t.Fatal(err)
				}
			}
			task := dsyOK(t, ts, "CreateTask", `{"SourceLocationArn":"`+src+`","DestinationLocationArn":"`+dst+`"}`)["TaskArn"].(string)
			got := dsyRun(t, ts, task, "")
			keys := dsyKeys(t, srv, "dsy-dst")
			if tc.principal == "" {
				if got["Status"] != "SUCCESS" || len(keys) != 1 {
					t.Fatalf("want SUCCESS with a copy, got %v %v", got, keys)
				}
				return
			}
			result, _ := got["Result"].(map[string]any)
			if got["Status"] != "ERROR" || result["ErrorCode"] != "AccessDenied" || len(keys) != 0 {
				t.Fatalf("want ERROR AccessDenied without a copy, got %v %v", got, keys)
			}
		})
	}
}

// Sync read with a store-held S3 service: Route 53 Resolver
// ImportFirewallDomains reads a domain file.
func TestRoute53ResolverServiceS3Authz(t *testing.T) {
	for _, tc := range svcAuthzCases("route53resolver.amazonaws.com") {
		t.Run(tc.name, func(t *testing.T) {
			srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
			ts := httptest.NewServer(srv.Handler())
			t.Cleanup(ts.Close)
			svcAuthzBucket(t, srv, "r53r-authz", tc.principal)
			if _, err := srv.s3.PutObject("r53r-authz", "domains.txt", "text/plain", []byte("example.com\n"), nil, "", ""); err != nil {
				t.Fatal(err)
			}
			call := func(op, body string) (int, map[string]any) { return r53rRequest(t, ts, op, body) }
			status, out := call("CreateFirewallDomainList", `{"CreatorRequestId":"r1","Name":"authz"}`)
			if status != http.StatusOK {
				t.Fatalf("create: %d %v", status, out)
			}
			id := out["FirewallDomainList"].(map[string]any)["Id"].(string)
			status, out = call("ImportFirewallDomains", `{"FirewallDomainListId":"`+id+`","Operation":"REPLACE","DomainFileUrl":"s3://r53r-authz/domains.txt"}`)
			if tc.principal == "" {
				if status != http.StatusOK {
					t.Fatalf("want 200, got %d %v", status, out)
				}
				return
			}
			if out["__type"] != "AccessDeniedException" {
				t.Fatalf("want AccessDeniedException, got %d %v", status, out)
			}
		})
	}
}

// An unsigned caller's ACL fallback must use the ACL of the version actually
// read, not the current version's.
func TestAuthorizeServiceS3VersionACL(t *testing.T) {
	srv, _ := newTextractTestServer(t)
	if _, err := srv.s3.CreateBucket("tx-ver"); err != nil {
		t.Fatal(err)
	}
	if err := srv.s3.SetBucketVersioning("tx-ver", "Enabled"); err != nil {
		t.Fatal(err)
	}
	old, err := srv.s3.PutObject("tx-ver", "doc.txt", "text/plain", []byte("private"), nil, "private", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.s3.PutObject("tx-ver", "doc.txt", "text/plain", []byte("public"), nil, "public-read", ""); err != nil {
		t.Fatal(err)
	}
	anon := s3Caller{}
	if !authorizeServiceS3Version(srv.s3, anon, txServicePrincipal, "tx-ver", "doc.txt", "", "s3:GetObject") {
		t.Fatal("current public-read version should be readable")
	}
	if authorizeServiceS3Version(srv.s3, anon, txServicePrincipal, "tx-ver", "doc.txt", old.VersionID, "s3:GetObject") {
		t.Fatal("private old version must not be readable through the current version's ACL")
	}
}

// A retried ClientRequestToken must not return the job to a caller who
// can't read its input.
func TestTextractJobTokenRetryRechecksInput(t *testing.T) {
	srv, ts := newTextractTestServer(t)
	svcAuthzBucket(t, srv, "tx-tok", "", "doc.txt")
	req := `{"ClientRequestToken":"tok-1","DocumentLocation":{"S3Object":{"Bucket":"tx-tok","Name":"doc.txt"}}}`
	txOK(t, ts, "StartDocumentTextDetection", req)
	if err := srv.s3.SetBucketPolicy("tx-tok", svcAuthzPolicy("Deny", `{"AWS":"`+testAccessKey+`"}`, "tx-tok")); err != nil {
		t.Fatal(err)
	}
	status, out := txCall(t, ts, "StartDocumentTextDetection", req)
	if status != http.StatusBadRequest || out["__type"] != "AccessDeniedException" {
		t.Fatalf("retry after deny: want AccessDeniedException, got %d %v", status, out)
	}
}

// Application Cost Profiler reads the usage file and writes reports on the
// caller's behalf.
func TestApplicationCostProfilerServiceS3Authz(t *testing.T) {
	for _, tc := range svcAuthzCases(acpServicePrincipal) {
		t.Run(tc.name, func(t *testing.T) {
			srv, ts := newBudTestServer(t)
			svcAuthzBucket(t, srv, "acp-reports", "")
			acpCall(t, ts, http.MethodPost, "/reportDefinition", acpDefJSON("daily", "acp-reports", "CSV", "DAILY"))
			usage := "ApplicationId,TenantId,TenantDesc,ResourceId,StartTime,EndTime\napp,a,Tenant A,i-1,2026-01-01T00:00:00Z,2026-01-01T03:00:00Z\n"
			svcAuthzBucket(t, srv, "acp-usage", tc.principal)
			if _, err := srv.s3.PutObject("acp-usage", "usage.csv", "text/csv", []byte(usage), nil, "", ""); err != nil {
				t.Fatal(err)
			}
			status, out := acpCall(t, ts, http.MethodPost, "/importApplicationUsage", `{"sourceS3Location":{"bucket":"acp-usage","key":"usage.csv"}}`)
			if tc.principal == "" {
				if status != http.StatusOK {
					t.Fatalf("want 200, got %d %v", status, out)
				}
				return
			}
			if status != http.StatusForbidden || out["__code"] != "AccessDeniedException" {
				t.Fatalf("want AccessDeniedException, got %d %v", status, out)
			}
		})
	}
}
