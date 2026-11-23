package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/services/privateca"
)

func pscTestCA(t *testing.T, srv *Server, usage string) string {
	t.Helper()
	arn, err := srv.privateca.CreateCertificateAuthority(privateca.CreateCertificateAuthorityInput{
		CertificateAuthorityType: "ROOT", UsageMode: usage,
		Configuration: privateca.CertificateAuthorityConfiguration{KeyAlgorithm: "RSA_2048", SigningAlgorithm: "SHA256WITHRSA"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return arn
}

func TestPCAConnectorSCEPStage1Connectors(t *testing.T) {
	srv, ts := newPCAConnectorSCEPTestServer(t)
	ca := pscTestCA(t, srv, "")

	// The CA must exist, be ACTIVE and general-purpose.
	missing := strings.Replace(ca, ca[len(ca)-4:], "ffff", 1)
	if code := pscErrCode(t, ts, http.MethodPost, "/connectors", fmt.Sprintf(`{"CertificateAuthorityArn":%q}`, missing)); code != "ResourceNotFoundException" {
		t.Fatalf("missing CA: %s", code)
	}
	if code := pscErrCode(t, ts, http.MethodPost, "/connectors", `{}`); code != "ValidationException" {
		t.Fatalf("no CA: %s", code)
	}
	shortLived := pscTestCA(t, srv, "SHORT_LIVED_CERTIFICATE")
	if code := pscErrCode(t, ts, http.MethodPost, "/connectors", fmt.Sprintf(`{"CertificateAuthorityArn":%q}`, shortLived)); code != "ValidationException" {
		t.Fatalf("short-lived CA: %s", code)
	}
	disabled := pscTestCA(t, srv, "")
	if err := srv.privateca.UpdateCertificateAuthority(privateca.UpdateCertificateAuthorityInput{ARN: disabled, Status: "DISABLED"}); err != nil {
		t.Fatal(err)
	}
	if code := pscErrCode(t, ts, http.MethodPost, "/connectors", fmt.Sprintf(`{"CertificateAuthorityArn":%q}`, disabled)); code != "ValidationException" {
		t.Fatalf("disabled CA: %s", code)
	}

	// Creation is idempotent on ClientToken; a new connector is CREATING, then ACTIVE.
	body := fmt.Sprintf(`{"CertificateAuthorityArn":%q,"ClientToken":"tok-1","Tags":{"env":"dev"}}`, ca)
	arn := pscCall(t, ts, http.StatusAccepted, http.MethodPost, "/connectors", body)["ConnectorArn"].(string)
	if again := pscCall(t, ts, http.StatusAccepted, http.MethodPost, "/connectors", body)["ConnectorArn"]; again != arn {
		t.Fatalf("idempotent create: %v vs %s", again, arn)
	}
	path := "/connectors/" + url.PathEscape(arn)
	c := pscCall(t, ts, http.StatusOK, http.MethodGet, path, "")["Connector"].(map[string]any)
	if c["Status"] != "ACTIVE" || c["Type"] != "GENERAL_PURPOSE" || !strings.Contains(c["Endpoint"].(string), "/_stackyard/pca-connector-scep/") {
		t.Fatalf("connector: %v", c)
	}

	// Intune connectors carry their configuration and an OpenID configuration.
	intune := pscCall(t, ts, http.StatusAccepted, http.MethodPost, "/connectors",
		fmt.Sprintf(`{"CertificateAuthorityArn":%q,"MobileDeviceManagement":{"Intune":{"AzureApplicationId":"aaaa-bbbb-cccc-dddd-eeee","Domain":"contoso.onmicrosoft.com"}}}`, ca))["ConnectorArn"].(string)
	ic := pscCall(t, ts, http.StatusOK, http.MethodGet, "/connectors/"+url.PathEscape(intune), "")["Connector"].(map[string]any)
	if ic["Type"] != "INTUNE" || ic["OpenIdConfiguration"] == nil {
		t.Fatalf("intune connector: %v", ic)
	}

	list := pscCall(t, ts, http.StatusOK, http.MethodGet, "/connectors?MaxResults=1", "")
	if len(list["Connectors"].([]any)) != 1 || list["NextToken"] == nil {
		t.Fatalf("page 1: %v", list)
	}
	list = pscCall(t, ts, http.StatusOK, http.MethodGet, "/connectors?MaxResults=1&NextToken="+url.QueryEscape(list["NextToken"].(string)), "")
	if len(list["Connectors"].([]any)) != 1 || list["NextToken"] != nil {
		t.Fatalf("page 2: %v", list)
	}
	if code := pscErrCode(t, ts, http.MethodGet, "/connectors?NextToken=bogus", ""); code != "ValidationException" {
		t.Fatalf("bad token: %s", code)
	}

	tags := pscCall(t, ts, http.StatusOK, http.MethodGet, "/tags/"+url.PathEscape(arn), "")["Tags"].(map[string]any)
	if tags["env"] != "dev" {
		t.Fatalf("tags: %v", tags)
	}
	pscCall(t, ts, http.StatusNoContent, http.MethodPost, "/tags/"+url.PathEscape(arn), `{"Tags":{"team":"core"}}`)
	if code := pscErrCode(t, ts, http.MethodPost, "/tags/"+url.PathEscape(arn), `{"Tags":{"aws:x":"y"}}`); code != "ValidationException" {
		t.Fatalf("aws: tag: %s", code)
	}
	pscCall(t, ts, http.StatusNoContent, http.MethodDelete, "/tags/"+url.PathEscape(arn)+"?tagKeys=env", "")
	tags = pscCall(t, ts, http.StatusOK, http.MethodGet, "/tags/"+url.PathEscape(arn), "")["Tags"].(map[string]any)
	if len(tags) != 1 || tags["team"] != "core" {
		t.Fatalf("tags after untag: %v", tags)
	}

	pscCall(t, ts, http.StatusAccepted, http.MethodDelete, path, "")
	if code := pscErrCode(t, ts, http.MethodGet, path, ""); code != "ResourceNotFoundException" {
		t.Fatalf("deleted connector: %s", code)
	}
}

func TestPCAConnectorSCEPStage1Challenges(t *testing.T) {
	srv, ts := newPCAConnectorSCEPTestServer(t)
	ca := pscTestCA(t, srv, "")
	arn := pscCall(t, ts, http.StatusAccepted, http.MethodPost, "/connectors", fmt.Sprintf(`{"CertificateAuthorityArn":%q}`, ca))["ConnectorArn"].(string)

	// A CREATING connector can't take challenges yet.
	if code := pscErrCode(t, ts, http.MethodPost, "/challenges", fmt.Sprintf(`{"ConnectorArn":%q}`, arn)); code != "ConflictException" {
		t.Fatalf("challenge on CREATING connector: %s", code)
	}
	pscCall(t, ts, http.StatusOK, http.MethodGet, "/connectors/"+url.PathEscape(arn), "")

	ch := pscCall(t, ts, http.StatusAccepted, http.MethodPost, "/challenges", fmt.Sprintf(`{"ConnectorArn":%q,"ClientToken":"c-1"}`, arn))["Challenge"].(map[string]any)
	chArn, password := ch["Arn"].(string), ch["Password"].(string)
	if !strings.HasPrefix(chArn, arn+"/challenge/") || len(password) < 16 {
		t.Fatalf("challenge: %v", ch)
	}
	// The password appears only in CreateChallenge and GetChallengePassword.
	meta := pscCall(t, ts, http.StatusOK, http.MethodGet, "/challengeMetadata/"+url.PathEscape(chArn), "")["ChallengeMetadata"].(map[string]any)
	if _, leaked := meta["Password"]; leaked || meta["ConnectorArn"] != arn {
		t.Fatalf("metadata: %v", meta)
	}
	list := pscCall(t, ts, http.StatusOK, http.MethodGet, "/challengeMetadata?ConnectorArn="+url.QueryEscape(arn), "")["Challenges"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["Password"] != nil {
		t.Fatalf("list: %v", list)
	}
	if got := pscCall(t, ts, http.StatusOK, http.MethodGet, "/challengePasswords/"+url.PathEscape(chArn), "")["Password"]; got != password {
		t.Fatalf("password: %v", got)
	}
	if code := pscErrCode(t, ts, http.MethodGet, "/challengeMetadata", ""); code != "ValidationException" {
		t.Fatalf("list without ConnectorArn: %s", code)
	}

	// Intune connectors don't use challenge passwords.
	intune := pscCall(t, ts, http.StatusAccepted, http.MethodPost, "/connectors",
		fmt.Sprintf(`{"CertificateAuthorityArn":%q,"MobileDeviceManagement":{"Intune":{"AzureApplicationId":"aaaa-bbbb-cccc-dddd-eeee","Domain":"contoso"}}}`, ca))["ConnectorArn"].(string)
	pscCall(t, ts, http.StatusOK, http.MethodGet, "/connectors/"+url.PathEscape(intune), "")
	if code := pscErrCode(t, ts, http.MethodPost, "/challenges", fmt.Sprintf(`{"ConnectorArn":%q}`, intune)); code != "BadRequestException" {
		t.Fatalf("challenge on Intune connector: %s", code)
	}

	pscCall(t, ts, http.StatusAccepted, http.MethodDelete, "/challenges/"+url.PathEscape(chArn), "")
	if code := pscErrCode(t, ts, http.MethodGet, "/challengePasswords/"+url.PathEscape(chArn), ""); code != "ResourceNotFoundException" {
		t.Fatalf("deleted challenge: %s", code)
	}

	// Deleting a connector deletes its challenges.
	ch2 := pscCall(t, ts, http.StatusAccepted, http.MethodPost, "/challenges", fmt.Sprintf(`{"ConnectorArn":%q}`, arn))["Challenge"].(map[string]any)["Arn"].(string)
	pscCall(t, ts, http.StatusAccepted, http.MethodDelete, "/connectors/"+url.PathEscape(arn), "")
	srv.pcaConnectorSCEP.mu.Lock()
	_, left := srv.pcaConnectorSCEP.challenges[ch2]
	srv.pcaConnectorSCEP.mu.Unlock()
	if left {
		t.Fatal("challenge survived its connector")
	}
}
