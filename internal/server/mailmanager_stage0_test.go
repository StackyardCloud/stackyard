package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newMessagingTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func msgDecode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("decode %q: %v", raw, err)
		}
	}
	return out
}

func mmgrCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": mmgrContentType, "X-Amz-Target": mmgrTargetPrefix + op,
	}, "ses")
	return resp.StatusCode, msgDecode(t, resp)
}

func mmgrOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := mmgrCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

// mmgrFail calls op expecting an error and returns its status and __type.
func mmgrFail(t *testing.T, ts *httptest.Server, op, body string) (int, string) {
	t.Helper()
	status, out := mmgrCall(t, ts, op, body)
	if status == http.StatusOK {
		t.Fatalf("%s: want an error, got %v", op, out)
	}
	code, _ := out["__type"].(string)
	return status, code
}

// msgREST calls a REST JSON path signed for service.
func msgREST(t *testing.T, ts *httptest.Server, service, method, path, body string) (int, map[string]any) {
	t.Helper()
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, map[string]string{"Content-Type": "application/json"}, service)
	return resp.StatusCode, msgDecode(t, resp)
}

func TestMessagingStage0CatalogsMatchModels(t *testing.T) {
	check := func(name string, model *awsmodels.Model, ops []string, handled func(string) bool) {
		if len(ops) != len(model.Operations) {
			t.Fatalf("%s: catalog has %d operations, model %d", name, len(ops), len(model.Operations))
		}
		for _, op := range ops {
			if _, ok := model.Operations[op]; !ok {
				t.Errorf("%s: %s is not in the model", name, op)
			}
			if !handled(op) {
				t.Errorf("%s: %s has no handler", name, op)
			}
		}
	}
	var ops []string
	for _, op := range mailManagerOperations {
		ops = append(ops, op.Name)
	}
	check("mailmanager", awsmodels.MailManagerModel(), ops, func(op string) bool { return mmgrHandlers[op] != nil })
	ops = nil
	for _, op := range pinpointEmailOperations {
		ops = append(ops, op.Name)
	}
	check("pinpoint-email", awsmodels.PinpointEmailModel(), ops, func(op string) bool { return ppeHandlers[op] != nil })
	ops = nil
	for _, op := range pinpointSMSVoiceOperations {
		ops = append(ops, op.Name)
	}
	check("pinpoint-sms-voice", awsmodels.PinpointSMSVoiceModel(), ops, func(string) bool { return true })
}

// Mail Manager, Pinpoint Email, SESv2 and SES share the ses signing name;
// Pinpoint SMS Voice and SMS Voice v2 share sms-voice. Each keeps its own
// traffic.
func TestMessagingStage0Routing(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	if out := mmgrOK(t, ts, "ListArchives", `{}`); out["Archives"] == nil {
		t.Fatalf("ListArchives: %v", out)
	}
	if status, out := msgREST(t, ts, "ses", http.MethodGet, "/v1/email/configuration-sets", ""); status != http.StatusOK || out["ConfigurationSets"] == nil {
		t.Fatalf("Pinpoint Email ListConfigurationSets: %d %v", status, out)
	}
	if status, out := msgREST(t, ts, "sms-voice", http.MethodGet, "/v1/sms-voice/configuration-sets", ""); status != http.StatusOK || out["ConfigurationSets"] == nil {
		t.Fatalf("Pinpoint SMS Voice ListConfigurationSets: %d %v", status, out)
	}
	// SESv2 keeps /v2/email and its own /v1/email/.../dkim/signing route.
	if status, out := msgREST(t, ts, "ses", http.MethodPost, "/v2/email/identities", `{"EmailIdentity":"example.com"}`); status != http.StatusOK || out["IdentityType"] != "DOMAIN" {
		t.Fatalf("SESv2 CreateEmailIdentity: %d %v", status, out)
	}
	if status, out := msgREST(t, ts, "ses", http.MethodPut, "/v1/email/identities/example.com/dkim/signing", `{"SigningAttributesOrigin":"AWS_SES"}`); status != http.StatusOK || out["DkimStatus"] == nil {
		t.Fatalf("SESv2 PutEmailIdentityDkimSigningAttributes: %d %v", status, out)
	}
	// SES (query API) keeps form posts to /.
	values := url.Values{"Action": {"ListIdentities"}}
	resp := sesRequest(t, ts, values)
	if body := string(mustBody(t, resp)); resp.StatusCode != http.StatusOK || !strings.Contains(body, "ListIdentitiesResponse") {
		t.Fatalf("SES ListIdentities: %d %s", resp.StatusCode, body)
	}
	// SMS Voice v2 keeps its X-Amz-Target traffic.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "PinpointSMSVoiceV2.DescribeConfigurationSets",
	}, "sms-voice")
	if out := msgDecode(t, resp); resp.StatusCode != http.StatusOK || out["ConfigurationSets"] == nil {
		t.Fatalf("SMS Voice v2 DescribeConfigurationSets: %d %v", resp.StatusCode, out)
	}
	// Pinpoint (mobiletargeting) keeps /v1/apps.
	if status, out := msgREST(t, ts, "mobiletargeting", http.MethodGet, "/v1/apps", ""); status != http.StatusOK || out["Item"] == nil {
		t.Fatalf("Pinpoint GetApps: %d %v", status, out)
	}
	// Requests signed for another service aren't claimed.
	if status, _ := msgREST(t, ts, "s3", http.MethodGet, "/v1/email/configuration-sets", ""); status == http.StatusOK {
		t.Fatalf("s3-signed /v1/email request was served")
	}
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": mmgrContentType, "X-Amz-Target": mmgrTargetPrefix + "ListArchives",
	}, "s3")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("s3-signed Mail Manager request: %d", resp.StatusCode)
	}
	resp.Body.Close()
	// Chunked bodies are refused.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", mmgrTargetPrefix+"ListArchives")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
	if status, code := mmgrFail(t, ts, "NoSuchOperation", `{}`); status != http.StatusBadRequest || code != "UnknownOperationException" {
		t.Fatalf("unknown op: %d %s", status, code)
	}
	if status, code := mmgrFail(t, ts, "CreateArchive", `{"ArchiveName":"-bad-"}`); status != http.StatusBadRequest || code != "ValidationException" {
		t.Fatalf("bad name: %d %s", status, code)
	}
	if status, code := mmgrFail(t, ts, "CreateArchive", `{"ArchiveName":"ok","Smuggled":1}`); status != http.StatusBadRequest || code != "ValidationException" {
		t.Fatalf("unknown member: %d %s", status, code)
	}
}
