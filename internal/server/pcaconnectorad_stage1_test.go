package server

import (
	"fmt"
	"net/http"
	"testing"
)

func TestPCAConnectorADStage1Lifecycle(t *testing.T) {
	srv, ts := newPCAConnectorADTestServer(t)
	ca := padNewCA(t, srv)

	// Directory registrations: the seeded directory is Active; a new one is
	// still Requested and settles to FAILED.
	reg := padCall(t, ts, http.StatusAccepted, "POST", "/directoryRegistrations", `{"DirectoryId":"d-0000000000","ClientToken":"reg-1"}`)["DirectoryRegistrationArn"].(string)
	if again := padCall(t, ts, http.StatusAccepted, "POST", "/directoryRegistrations", `{"DirectoryId":"d-0000000000","ClientToken":"reg-1"}`)["DirectoryRegistrationArn"]; again != reg {
		t.Fatalf("client token: %v != %s", again, reg)
	}
	if code, _ := padErr(t, ts, "POST", "/directoryRegistrations", `{"DirectoryId":"d-0000000000"}`); code != "ConflictException" {
		t.Fatalf("duplicate registration: %s", code)
	}
	if code, _ := padErr(t, ts, "POST", "/directoryRegistrations", `{"DirectoryId":"d-0123456789"}`); code != "ResourceNotFoundException" {
		t.Fatalf("unknown directory: %s", code)
	}
	path := "/directoryRegistrations/" + padEsc(reg)
	if st := padCall(t, ts, 200, "GET", path, "")["DirectoryRegistration"].(map[string]any)["Status"]; st != "CREATING" {
		t.Fatalf("first read: %v", st)
	}
	if st := padCall(t, ts, 200, "GET", path, "")["DirectoryRegistration"].(map[string]any)["Status"]; st != "ACTIVE" {
		t.Fatalf("second read: %v", st)
	}
	otherDir := srv.directoryservice.Handle("CreateDirectory", map[string]any{"Name": "other.local"})["DirectoryId"].(string)
	otherReg := padCall(t, ts, http.StatusAccepted, "POST", "/directoryRegistrations", fmt.Sprintf(`{"DirectoryId":%q}`, otherDir))["DirectoryRegistrationArn"].(string)
	padCall(t, ts, 200, "GET", "/directoryRegistrations/"+padEsc(otherReg), "")
	failed := padCall(t, ts, 200, "GET", "/directoryRegistrations/"+padEsc(otherReg), "")["DirectoryRegistration"].(map[string]any)
	if failed["Status"] != "FAILED" || failed["StatusReason"] != "DIRECTORY_NOT_ACTIVE" {
		t.Fatalf("registration of a Requested directory: %v", failed)
	}
	page := padCall(t, ts, 200, "GET", "/directoryRegistrations?MaxResults=1", "")
	if len(page["DirectoryRegistrations"].([]any)) != 1 || page["NextToken"] == nil {
		t.Fatalf("paging: %v", page)
	}

	// Connectors: the CA, directory and security groups must exist.
	connBody := func(caARN, dir, sg string) string {
		return fmt.Sprintf(`{"CertificateAuthorityArn":%q,"DirectoryId":%q,"VpcInformation":{"SecurityGroupIds":[%q]}}`, caARN, dir, sg)
	}
	missingCA := "arn:aws:acm-pca:us-east-1:123456789012:certificate-authority/00000000-0000-0000-0000-000000000000"
	if code, _ := padErr(t, ts, "POST", "/connectors", connBody(missingCA, "d-0000000000", "sg-00000000")); code != "ResourceNotFoundException" {
		t.Fatalf("missing CA: %s", code)
	}
	if code, reason := padErr(t, ts, "POST", "/connectors", connBody(ca, "d-0000000000", "sg-0badbeef")); code != "ValidationException" || reason != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("missing security group: %s %s", code, reason)
	}
	conn := padCall(t, ts, http.StatusAccepted, "POST", "/connectors", connBody(ca, "d-0000000000", "sg-00000000"))["ConnectorArn"].(string)
	got := padCall(t, ts, 200, "GET", "/connectors/"+padEsc(conn), "")["Connector"].(map[string]any)
	if got["Status"] != "CREATING" || got["VpcInformation"].(map[string]any)["IpAddressType"] != "IPV4" {
		t.Fatalf("connector: %v", got)
	}
	otherConn := padCall(t, ts, http.StatusAccepted, "POST", "/connectors", connBody(ca, otherDir, "sg-00000000"))["ConnectorArn"].(string)

	// Service principal names join a registration and a connector for the same directory.
	spn := "/directoryRegistrations/" + padEsc(reg) + "/servicePrincipalNames/"
	padCall(t, ts, http.StatusAccepted, "POST", spn+padEsc(conn), `{}`)
	if code, _ := padErr(t, ts, "POST", spn+padEsc(conn), `{}`); code != "ConflictException" {
		t.Fatalf("duplicate SPN: %s", code)
	}
	if code, reason := padErr(t, ts, "POST", spn+padEsc(otherConn), `{}`); code != "ValidationException" || reason != "MISMATCHED_CONNECTOR" {
		t.Fatalf("mismatched connector: %s %s", code, reason)
	}
	if n := len(padCall(t, ts, 200, "GET", "/directoryRegistrations/"+padEsc(reg)+"/servicePrincipalNames", "")["ServicePrincipalNames"].([]any)); n != 1 {
		t.Fatalf("SPN list: %d", n)
	}
	if code, _ := padErr(t, ts, "DELETE", "/connectors/"+padEsc(conn), ""); code != "ConflictException" {
		t.Fatalf("deleting a connector in use: %s", code)
	}
	if code, _ := padErr(t, ts, "DELETE", "/directoryRegistrations/"+padEsc(reg), ""); code != "ConflictException" {
		t.Fatalf("deleting a registration in use: %s", code)
	}

	// Templates.
	tmplBody := func(name string) string {
		return fmt.Sprintf(`{"ConnectorArn":%q,"Name":%q,"Definition":%s}`, conn, name, padTestDefinitionJSON)
	}
	tmpl := padCall(t, ts, 200, "POST", "/templates", tmplBody("WebServer"))["TemplateArn"].(string)
	if code, _ := padErr(t, ts, "POST", "/templates", tmplBody("WebServer")); code != "ConflictException" {
		t.Fatalf("duplicate template name: %s", code)
	}
	if code, _ := padErr(t, ts, "POST", "/templates", tmplBody("bad;name")); code != "ValidationException" {
		t.Fatalf("invalid template name: %s", code)
	}
	tp := "/templates/" + padEsc(tmpl)
	padCall(t, ts, 200, "PATCH", tp, `{"Definition":`+padTestDefinitionJSON+`}`)
	padCall(t, ts, 200, "PATCH", tp, `{"ReenrollAllCertificateHolders":true}`)
	got = padCall(t, ts, 200, "GET", tp, "")["Template"].(map[string]any)
	if rev := got["Revision"].(map[string]any); rev["MajorRevision"] != float64(2) || rev["MinorRevision"] != float64(1) || got["PolicySchema"] != float64(2) {
		t.Fatalf("template revision: %v", got)
	}
	if n := len(padCall(t, ts, 200, "GET", "/templates?ConnectorArn="+padEsc(conn), "")["Templates"].([]any)); n != 1 {
		t.Fatalf("template list: %d", n)
	}

	// Group access control entries.
	ace := tp + "/accessControlEntries"
	sid := "S-1-5-21-1004336348-1177238915-682003330-512"
	padCall(t, ts, 200, "POST", ace, fmt.Sprintf(`{"GroupSecurityIdentifier":%q,"GroupDisplayName":"Domain Admins","AccessRights":{"Enroll":"ALLOW"}}`, sid))
	if code, _ := padErr(t, ts, "POST", ace, fmt.Sprintf(`{"GroupSecurityIdentifier":%q,"GroupDisplayName":"x","AccessRights":{"Enroll":"ALLOW"}}`, sid)); code != "ConflictException" {
		t.Fatalf("duplicate ACE: %s", code)
	}
	padCall(t, ts, 200, "PATCH", ace+"/"+sid, `{"AccessRights":{"AutoEnroll":"DENY"}}`)
	rights := padCall(t, ts, 200, "GET", ace+"/"+sid, "")["AccessControlEntry"].(map[string]any)["AccessRights"].(map[string]any)
	if rights["Enroll"] != "ALLOW" || rights["AutoEnroll"] != "DENY" {
		t.Fatalf("ACE update should merge rights: %v", rights)
	}
	padCall(t, ts, 200, "DELETE", ace+"/"+sid, "")
	if code, _ := padErr(t, ts, "GET", ace+"/"+sid, ""); code != "ResourceNotFoundException" {
		t.Fatalf("deleted ACE: %s", code)
	}

	// Tags.
	tags := "/tags/" + padEsc(conn)
	padCall(t, ts, http.StatusNoContent, "POST", tags, `{"Tags":{"env":"dev","team":"pki"}}`)
	padCall(t, ts, http.StatusNoContent, "DELETE", tags+"?tagKeys=team", "")
	if got := padCall(t, ts, 200, "GET", tags, "")["Tags"].(map[string]any); len(got) != 1 || got["env"] != "dev" {
		t.Fatalf("tags: %v", got)
	}
	if code, _ := padErr(t, ts, "GET", "/tags/"+padEsc(missingCA), ""); code != "ResourceNotFoundException" {
		t.Fatalf("tags of an unknown resource: %s", code)
	}

	// Deleting the SPN frees the connector; its templates go with it.
	padCall(t, ts, http.StatusAccepted, "DELETE", spn+padEsc(conn), "")
	padCall(t, ts, http.StatusAccepted, "DELETE", "/connectors/"+padEsc(conn), "")
	if code, _ := padErr(t, ts, "GET", tp, ""); code != "ResourceNotFoundException" {
		t.Fatalf("template of a deleted connector: %s", code)
	}
	padCall(t, ts, http.StatusAccepted, "DELETE", "/directoryRegistrations/"+padEsc(reg), "")
}
