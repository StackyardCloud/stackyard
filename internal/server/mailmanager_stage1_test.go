package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// SES sends through a configuration set with archiving options land in the
// Mail Manager archive, where they can be searched, read, downloaded and
// exported to S3.
func TestMailManagerStage1ArchiveFromSES(t *testing.T) {
	srv, ts := newMessagingTestServer(t)
	archive := mmgrOK(t, ts, "CreateArchive", `{"ArchiveName":"inbox","Tags":[{"Key":"team","Value":"mail"}]}`)["ArchiveId"].(string)
	got := mmgrOK(t, ts, "GetArchive", fmt.Sprintf(`{"ArchiveId":%q}`, archive))
	if got["ArchiveState"] != "ACTIVE" || got["Retention"].(map[string]any)["RetentionPeriod"] != "THREE_MONTHS" {
		t.Fatalf("GetArchive: %v", got)
	}
	arn := got["ArchiveArn"].(string)
	if status, code := mmgrFail(t, ts, "CreateArchive", `{"ArchiveName":"inbox"}`); status != http.StatusConflict || code != "ConflictException" {
		t.Fatalf("duplicate archive name: %d %s", status, code)
	}

	for _, step := range []struct{ method, path, body string }{
		{http.MethodPost, "/v2/email/identities", `{"EmailIdentity":"example.com"}`},
		{http.MethodPost, "/v2/email/configuration-sets", `{"ConfigurationSetName":"archived"}`},
		{http.MethodPut, "/v2/email/configuration-sets/archived/archiving-options", fmt.Sprintf(`{"ArchiveArn":%q}`, arn)},
	} {
		if status, out := msgREST(t, ts, "ses", step.method, step.path, step.body); status != http.StatusOK {
			t.Fatalf("%s %s: %d %v", step.method, step.path, status, out)
		}
	}
	if _, out := msgREST(t, ts, "ses", http.MethodGet, "/v2/email/configuration-sets/archived", ""); out["ArchivingOptions"].(map[string]any)["ArchiveArn"] != arn {
		t.Fatalf("SESv2 GetConfigurationSet: %v", out)
	}
	send := func(subject string) {
		body := fmt.Sprintf(`{"FromEmailAddress":"news@example.com","ConfigurationSetName":"archived",
			"Destination":{"ToAddresses":["ada@example.org"],"CcAddresses":["bob@example.org"]},
			"Content":{"Simple":{"Subject":{"Data":%q},"Body":{"Text":{"Data":"hello"},"Html":{"Data":"<p>hello</p>"}}}}}`, subject)
		if status, out := msgREST(t, ts, "ses", http.MethodPost, "/v2/email/outbound-emails", body); status != http.StatusOK {
			t.Fatalf("SESv2 SendEmail: %d %v", status, out)
		}
	}
	send("Weekly digest")
	send("Invoice ready")
	// Pinpoint Email sends through the same configuration set are archived too.
	if status, out := msgREST(t, ts, "ses", http.MethodPost, "/v1/email/outbound-emails", `{"FromEmailAddress":"news@example.com","ConfigurationSetName":"archived",
		"Destination":{"ToAddresses":["carol@example.org"]},"Content":{"Simple":{"Subject":{"Data":"From Pinpoint"},"Body":{"Text":{"Data":"hi"}}}}}`); status != http.StatusOK {
		t.Fatalf("Pinpoint Email SendEmail: %d %v", status, out)
	}

	window := fmt.Sprintf(`"FromTimestamp":%d,"ToTimestamp":%d`, time.Now().Add(-time.Hour).Unix(), time.Now().Add(time.Hour).Unix())
	search := mmgrOK(t, ts, "StartArchiveSearch", fmt.Sprintf(`{"ArchiveId":%q,%s,"MaxResults":10,
		"Filters":{"Include":[{"StringExpression":{"Evaluate":{"Attribute":"SUBJECT"},"Operator":"CONTAINS","Values":["invoice"]}}]}}`, archive, window))["SearchId"].(string)
	rows := mmgrOK(t, ts, "GetArchiveSearchResults", fmt.Sprintf(`{"SearchId":%q}`, search))["Rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("search rows: %v", rows)
	}
	row := rows[0].(map[string]any)
	if row["Subject"] != "Invoice ready" || row["From"] != "news@example.com" || row["Cc"] != "bob@example.org" {
		t.Fatalf("row: %v", row)
	}
	if st := mmgrOK(t, ts, "GetArchiveSearch", fmt.Sprintf(`{"SearchId":%q}`, search))["Status"].(map[string]any); st["State"] != "COMPLETED" {
		t.Fatalf("search status: %v", st)
	}
	all := mmgrOK(t, ts, "StartArchiveSearch", fmt.Sprintf(`{"ArchiveId":%q,%s,"MaxResults":10,
		"Filters":{"Unless":[{"StringExpression":{"Evaluate":{"Attribute":"TO"},"Operator":"CONTAINS","Values":["carol"]}}]}}`, archive, window))["SearchId"].(string)
	if rows := mmgrOK(t, ts, "GetArchiveSearchResults", fmt.Sprintf(`{"SearchId":%q}`, all))["Rows"].([]any); len(rows) != 2 {
		t.Fatalf("Unless filter rows: %d", len(rows))
	}
	// A search stopped before it settles is cancelled and has no results.
	stopped := mmgrOK(t, ts, "StartArchiveSearch", fmt.Sprintf(`{"ArchiveId":%q,%s,"MaxResults":10}`, archive, window))["SearchId"].(string)
	mmgrOK(t, ts, "StopArchiveSearch", fmt.Sprintf(`{"SearchId":%q}`, stopped))
	if status, code := mmgrFail(t, ts, "GetArchiveSearchResults", fmt.Sprintf(`{"SearchId":%q}`, stopped)); status != http.StatusConflict || code != "ConflictException" {
		t.Fatalf("results of a cancelled search: %d %s", status, code)
	}
	if status, _ := mmgrFail(t, ts, "StartArchiveSearch", fmt.Sprintf(`{"ArchiveId":%q,"FromTimestamp":10,"ToTimestamp":5,"MaxResults":1}`, archive)); status != http.StatusBadRequest {
		t.Fatalf("inverted window: %d", status)
	}

	msgID := row["ArchivedMessageId"].(string)
	content := mmgrOK(t, ts, "GetArchiveMessageContent", fmt.Sprintf(`{"ArchivedMessageId":%q}`, msgID))["Body"].(map[string]any)
	if content["Text"] != "hello" || content["Html"] != "<p>hello</p>" {
		t.Fatalf("content: %v", content)
	}
	msg := mmgrOK(t, ts, "GetArchiveMessage", fmt.Sprintf(`{"ArchivedMessageId":%q}`, msgID))
	if msg["Metadata"].(map[string]any)["ConfigurationSet"] != "archived" {
		t.Fatalf("metadata: %v", msg)
	}
	if to := msg["Envelope"].(map[string]any)["To"].([]any); len(to) != 2 {
		t.Fatalf("envelope: %v", msg["Envelope"])
	}
	dl, err := http.Get(msg["MessageDownloadLink"].(string))
	if err != nil {
		t.Fatal(err)
	}
	eml, _ := io.ReadAll(dl.Body)
	dl.Body.Close()
	if dl.StatusCode != http.StatusOK || dl.Header.Get("Content-Type") != "message/rfc822" || dl.Header.Get("X-Content-Type-Options") != "nosniff" || !bytes.Contains(eml, []byte("Subject: Invoice ready")) {
		t.Fatalf("download: %d %v %s", dl.StatusCode, dl.Header, eml)
	}

	// Export writes .eml files to Stackyard's S3; a missing bucket fails the export.
	if _, err := srv.s3.CreateBucket("mail-exports"); err != nil {
		t.Fatal(err)
	}
	export := mmgrOK(t, ts, "StartArchiveExport", fmt.Sprintf(`{"ArchiveId":%q,%s,"IncludeMetadata":true,
		"ExportDestinationConfiguration":{"S3":{"S3Location":"s3://mail-exports/out"}}}`, archive, window))["ExportId"].(string)
	if st := mmgrOK(t, ts, "GetArchiveExport", fmt.Sprintf(`{"ExportId":%q}`, export))["Status"].(map[string]any); st["State"] != "COMPLETED" {
		t.Fatalf("export status: %v", st)
	}
	objs, err := srv.s3.ListObjects("mail-exports", "out/"+export+"/")
	if err != nil || len(objs) != 6 {
		t.Fatalf("exported objects: %d %v", len(objs), err)
	}
	failed := mmgrOK(t, ts, "StartArchiveExport", fmt.Sprintf(`{"ArchiveId":%q,%s,
		"ExportDestinationConfiguration":{"S3":{"S3Location":"s3://no-such-bucket/out"}}}`, archive, window))["ExportId"].(string)
	if st := mmgrOK(t, ts, "GetArchiveExport", fmt.Sprintf(`{"ExportId":%q}`, failed))["Status"].(map[string]any); st["State"] != "FAILED" || st["ErrorMessage"] == nil {
		t.Fatalf("failed export status: %v", st)
	}
	if exports := mmgrOK(t, ts, "ListArchiveExports", fmt.Sprintf(`{"ArchiveId":%q}`, archive))["Exports"].([]any); len(exports) != 2 {
		t.Fatalf("ListArchiveExports: %v", exports)
	}

	// Deleting the archive leaves it PENDING_DELETION with its messages gone.
	mmgrOK(t, ts, "DeleteArchive", fmt.Sprintf(`{"ArchiveId":%q}`, archive))
	if got := mmgrOK(t, ts, "GetArchive", fmt.Sprintf(`{"ArchiveId":%q}`, archive)); got["ArchiveState"] != "PENDING_DELETION" {
		t.Fatalf("deleted archive: %v", got)
	}
	if status, _ := mmgrFail(t, ts, "GetArchiveMessage", fmt.Sprintf(`{"ArchivedMessageId":%q}`, msgID)); status != http.StatusNotFound {
		t.Fatalf("message of a deleted archive: %d", status)
	}
	if status, code := mmgrFail(t, ts, "UpdateArchive", fmt.Sprintf(`{"ArchiveId":%q,"ArchiveName":"renamed"}`, archive)); status != http.StatusConflict || code != "ConflictException" {
		t.Fatalf("update of a deleted archive: %d %s", status, code)
	}
}

// Rule sets and traffic policies reference archives, relays and address
// lists; ingress points reference both. References must exist, and
// referenced resources can't be deleted.
func TestMailManagerStage1References(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	archive := mmgrOK(t, ts, "CreateArchive", `{"ArchiveName":"kept"}`)["ArchiveId"].(string)
	relay := mmgrOK(t, ts, "CreateRelay", `{"RelayName":"smarthost","ServerName":"smtp.example.com","ServerPort":587,"Authentication":{"NoAuthentication":{}}}`)["RelayId"].(string)
	list := mmgrOK(t, ts, "CreateAddressList", `{"AddressListName":"vip"}`)["AddressListId"].(string)
	listARN := mmgrOK(t, ts, "GetAddressList", fmt.Sprintf(`{"AddressListId":%q}`, list))["AddressListArn"].(string)
	if status, code := mmgrFail(t, ts, "CreateRelay", `{"RelayName":"both","ServerName":"smtp.example.com","ServerPort":25,"Authentication":{"SecretArn":"arn:aws:secretsmanager:us-east-1:123456789012:secret:x","NoAuthentication":{}}}`); status != http.StatusBadRequest || code != "ValidationException" {
		t.Fatalf("relay with two authentications: %d %s", status, code)
	}

	rules := fmt.Sprintf(`[{"Name":"keep","Conditions":[{"BooleanExpression":{"Evaluate":{"IsInAddressList":{"Attribute":"RECIPIENT","AddressLists":[%q]}},"Operator":"IS_TRUE"}}],
		"Actions":[{"Archive":{"TargetArchive":%q}},{"Relay":{"Relay":%q}}]}]`, listARN, archive, relay)
	if status, code := mmgrFail(t, ts, "CreateRuleSet", `{"RuleSetName":"bad","Rules":[{"Actions":[{"Archive":{"TargetArchive":"a-missing"}}]}]}`); status != http.StatusBadRequest || code != "ValidationException" {
		t.Fatalf("rule set with a missing archive: %d %s", status, code)
	}
	ruleSet := mmgrOK(t, ts, "CreateRuleSet", fmt.Sprintf(`{"RuleSetName":"main","Rules":%s,"ClientToken":"tok-1"}`, rules))["RuleSetId"].(string)
	if again := mmgrOK(t, ts, "CreateRuleSet", fmt.Sprintf(`{"RuleSetName":"main","Rules":%s,"ClientToken":"tok-1"}`, rules))["RuleSetId"]; again != ruleSet {
		t.Fatalf("idempotent retry: %v != %s", again, ruleSet)
	}
	got := mmgrOK(t, ts, "GetRuleSet", fmt.Sprintf(`{"RuleSetId":%q}`, ruleSet))
	if len(got["Rules"].([]any)[0].(map[string]any)["Actions"].([]any)) != 2 {
		t.Fatalf("GetRuleSet: %v", got)
	}
	policy := mmgrOK(t, ts, "CreateTrafficPolicy", `{"TrafficPolicyName":"default-deny","DefaultAction":"DENY","MaxMessageSizeBytes":1000000,
		"PolicyStatements":[{"Action":"ALLOW","Conditions":[{"IpExpression":{"Evaluate":{"Attribute":"SENDER_IP"},"Operator":"CIDR_MATCHES","Values":["10.0.0.0/8"]}}]}]}`)["TrafficPolicyId"].(string)

	for _, op := range []struct{ op, body string }{
		{"DeleteArchive", fmt.Sprintf(`{"ArchiveId":%q}`, archive)},
		{"DeleteRelay", fmt.Sprintf(`{"RelayId":%q}`, relay)},
		{"DeleteAddressList", fmt.Sprintf(`{"AddressListId":%q}`, list)},
	} {
		if status, code := mmgrFail(t, ts, op.op, op.body); status != http.StatusConflict || code != "ConflictException" {
			t.Fatalf("%s of a referenced resource: %d %s", op.op, status, code)
		}
	}

	if status, code := mmgrFail(t, ts, "CreateIngressPoint", fmt.Sprintf(`{"IngressPointName":"smtp-auth","Type":"AUTH","RuleSetId":%q,"TrafficPolicyId":%q}`, ruleSet, policy)); status != http.StatusBadRequest || code != "ValidationException" {
		t.Fatalf("AUTH ingress point without a password: %d %s", status, code)
	}
	ingress := mmgrOK(t, ts, "CreateIngressPoint", fmt.Sprintf(`{"IngressPointName":"smtp-auth","Type":"AUTH","RuleSetId":%q,"TrafficPolicyId":%q,
		"IngressPointConfiguration":{"SmtpPassword":"correct-horse-battery-staple"}}`, ruleSet, policy))["IngressPointId"].(string)
	mmgrOK(t, ts, "UpdateIngressPoint", fmt.Sprintf(`{"IngressPointId":%q,"IngressPointConfiguration":{"SmtpPassword":"second-password-value"},"StatusToUpdate":"CLOSED"}`, ingress))
	status, raw := mmgrCall(t, ts, "GetIngressPoint", fmt.Sprintf(`{"IngressPointId":%q}`, ingress))
	if status != http.StatusOK || strings.Contains(fmt.Sprint(raw), "password-value") || strings.Contains(fmt.Sprint(raw), "battery") {
		t.Fatalf("GetIngressPoint leaked a password or failed: %d %v", status, raw)
	}
	pw := raw["IngressPointAuthConfiguration"].(map[string]any)["IngressPointPasswordConfiguration"].(map[string]any)
	if raw["Status"] != "CLOSED" || pw["SmtpPasswordVersion"] != "v2" || pw["PreviousSmtpPasswordVersion"] != "v1" || raw["ARecord"] == nil {
		t.Fatalf("GetIngressPoint: %v", raw)
	}
	if status, code := mmgrFail(t, ts, "DeleteRuleSet", fmt.Sprintf(`{"RuleSetId":%q}`, ruleSet)); status != http.StatusConflict || code != "ConflictException" {
		t.Fatalf("delete of a rule set in use: %d %s", status, code)
	}
	mmgrOK(t, ts, "DeleteIngressPoint", fmt.Sprintf(`{"IngressPointId":%q}`, ingress))
	mmgrOK(t, ts, "DeleteRuleSet", fmt.Sprintf(`{"RuleSetId":%q}`, ruleSet))
	mmgrOK(t, ts, "DeleteRelay", fmt.Sprintf(`{"RelayId":%q}`, relay))
	mmgrOK(t, ts, "DeleteTrafficPolicy", fmt.Sprintf(`{"TrafficPolicyId":%q}`, policy))
	if status, code := mmgrFail(t, ts, "GetRelay", fmt.Sprintf(`{"RelayId":%q}`, relay)); status != http.StatusNotFound || code != "ResourceNotFoundException" {
		t.Fatalf("deleted relay: %d %s", status, code)
	}

	// Tags, add-ons and paging.
	arn := mmgrOK(t, ts, "GetArchive", fmt.Sprintf(`{"ArchiveId":%q}`, archive))["ArchiveArn"].(string)
	mmgrOK(t, ts, "TagResource", fmt.Sprintf(`{"ResourceArn":%q,"Tags":[{"Key":"a","Value":"1"},{"Key":"b","Value":"2"}]}`, arn))
	mmgrOK(t, ts, "UntagResource", fmt.Sprintf(`{"ResourceArn":%q,"TagKeys":["a"]}`, arn))
	if tags := mmgrOK(t, ts, "ListTagsForResource", fmt.Sprintf(`{"ResourceArn":%q}`, arn))["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	if status, _ := mmgrFail(t, ts, "TagResource", `{"ResourceArn":"arn:aws:ses:us-east-1:123456789012:mailmanager-archive/a-missing","Tags":[{"Key":"a","Value":"1"}]}`); status != http.StatusNotFound {
		t.Fatalf("tag a missing resource: %d", status)
	}
	sub := mmgrOK(t, ts, "CreateAddonSubscription", `{"AddonName":"SPAMHAUS_DBL"}`)["AddonSubscriptionId"].(string)
	inst := mmgrOK(t, ts, "CreateAddonInstance", fmt.Sprintf(`{"AddonSubscriptionId":%q}`, sub))["AddonInstanceId"].(string)
	if got := mmgrOK(t, ts, "GetAddonInstance", fmt.Sprintf(`{"AddonInstanceId":%q}`, inst)); got["AddonName"] != "SPAMHAUS_DBL" {
		t.Fatalf("GetAddonInstance: %v", got)
	}
	if status, _ := mmgrFail(t, ts, "DeleteAddonSubscription", fmt.Sprintf(`{"AddonSubscriptionId":%q}`, sub)); status != http.StatusConflict {
		t.Fatalf("delete a subscription in use: %d", status)
	}
	for i := 0; i < 3; i++ {
		mmgrOK(t, ts, "CreateAddressList", fmt.Sprintf(`{"AddressListName":"page-%d"}`, i))
	}
	page := mmgrOK(t, ts, "ListAddressLists", `{"PageSize":2}`)
	next := mmgrOK(t, ts, "ListAddressLists", fmt.Sprintf(`{"PageSize":2,"NextToken":%q}`, page["NextToken"]))
	if len(page["AddressLists"].([]any)) != 2 || len(next["AddressLists"].([]any)) != 2 || next["NextToken"] != nil {
		t.Fatalf("paging: %v / %v", page, next)
	}
	if status, _ := mmgrFail(t, ts, "ListAddressLists", `{"NextToken":"99x"}`); status != http.StatusBadRequest {
		t.Fatalf("bad token: %d", status)
	}
}

// Address list members come from the API and from CSV or JSON uploads to an
// import job's pre-signed URL.
func TestMailManagerStage1AddressListImport(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	list := mmgrOK(t, ts, "CreateAddressList", `{"AddressListName":"customers"}`)["AddressListId"].(string)
	mmgrOK(t, ts, "RegisterMemberToAddressList", fmt.Sprintf(`{"AddressListId":%q,"Address":"Ada@Example.com"}`, list))
	if status, code := mmgrFail(t, ts, "RegisterMemberToAddressList", fmt.Sprintf(`{"AddressListId":%q,"Address":"ada@example.com"}`, list)); status != http.StatusConflict || code != "ConflictException" {
		t.Fatalf("duplicate member: %d %s", status, code)
	}
	if status, _ := mmgrFail(t, ts, "RegisterMemberToAddressList", fmt.Sprintf(`{"AddressListId":%q,"Address":"not-an-address"}`, list)); status != http.StatusBadRequest {
		t.Fatalf("invalid member: %d", status)
	}
	job := mmgrOK(t, ts, "CreateAddressListImportJob", fmt.Sprintf(`{"AddressListId":%q,"Name":"bulk","ImportDataFormat":{"ImportDataType":"CSV"}}`, list))
	upload := func(u, body string) int {
		req, _ := http.NewRequest(http.MethodPut, u, strings.NewReader(body))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	presigned := job["PreSignedUrl"].(string)
	if code := upload(strings.Split(presigned, "?")[0]+"?key=wrong", "address\n"); code != http.StatusForbidden {
		t.Fatalf("upload with a wrong key: %d", code)
	}
	if code := upload(presigned, "name,address\nBob,bob@example.com\nEve,not-valid\nAda,ada@example.com\n"); code != http.StatusOK {
		t.Fatalf("upload: %d", code)
	}
	jobID := job["JobId"].(string)
	mmgrOK(t, ts, "StartAddressListImportJob", fmt.Sprintf(`{"JobId":%q}`, jobID))
	got := mmgrOK(t, ts, "GetAddressListImportJob", fmt.Sprintf(`{"JobId":%q}`, jobID))
	if got["Status"] != "COMPLETED" || got["ImportedItemsCount"] != 2.0 || got["FailedItemsCount"] != 1.0 {
		t.Fatalf("import job: %v", got)
	}
	if code := upload(presigned, "address\nlate@example.com\n"); code != http.StatusConflict {
		t.Fatalf("upload after start: %d", code)
	}
	members := mmgrOK(t, ts, "ListMembersOfAddressList", fmt.Sprintf(`{"AddressListId":%q,"Filter":{"AddressPrefix":"b"}}`, list))["Addresses"].([]any)
	if len(members) != 1 || members[0].(map[string]any)["Address"] != "bob@example.com" {
		t.Fatalf("members: %v", members)
	}
	mmgrOK(t, ts, "GetMemberOfAddressList", fmt.Sprintf(`{"AddressListId":%q,"Address":"bob@example.com"}`, list))
	mmgrOK(t, ts, "DeregisterMemberFromAddressList", fmt.Sprintf(`{"AddressListId":%q,"Address":"bob@example.com"}`, list))
	if status, _ := mmgrFail(t, ts, "GetMemberOfAddressList", fmt.Sprintf(`{"AddressListId":%q,"Address":"bob@example.com"}`, list)); status != http.StatusNotFound {
		t.Fatalf("deregistered member: %d", status)
	}

	// A job started without an upload fails; a stopped job never applies.
	empty := mmgrOK(t, ts, "CreateAddressListImportJob", fmt.Sprintf(`{"AddressListId":%q,"Name":"empty","ImportDataFormat":{"ImportDataType":"JSON"}}`, list))["JobId"].(string)
	mmgrOK(t, ts, "StartAddressListImportJob", fmt.Sprintf(`{"JobId":%q}`, empty))
	if got := mmgrOK(t, ts, "GetAddressListImportJob", fmt.Sprintf(`{"JobId":%q}`, empty)); got["Status"] != "FAILED" || got["Error"] == nil {
		t.Fatalf("empty job: %v", got)
	}
	stop := mmgrOK(t, ts, "CreateAddressListImportJob", fmt.Sprintf(`{"AddressListId":%q,"Name":"stop","ImportDataFormat":{"ImportDataType":"JSON"}}`, list))
	if code := upload(stop["PreSignedUrl"].(string), `{"address":"zed@example.com"}`); code != http.StatusOK {
		t.Fatalf("JSON upload: %d", code)
	}
	if status, code := mmgrFail(t, ts, "StopAddressListImportJob", fmt.Sprintf(`{"JobId":%q}`, stop["JobId"])); status != http.StatusConflict || code != "ConflictException" {
		t.Fatalf("stop a job that hasn't started: %d %s", status, code)
	}
	mmgrOK(t, ts, "StartAddressListImportJob", fmt.Sprintf(`{"JobId":%q}`, stop["JobId"]))
	mmgrOK(t, ts, "StopAddressListImportJob", fmt.Sprintf(`{"JobId":%q}`, stop["JobId"]))
	if got := mmgrOK(t, ts, "GetAddressListImportJob", fmt.Sprintf(`{"JobId":%q}`, stop["JobId"])); got["Status"] != "STOPPED" {
		t.Fatalf("stopped job: %v", got)
	}
	if status, _ := mmgrFail(t, ts, "GetMemberOfAddressList", fmt.Sprintf(`{"AddressListId":%q,"Address":"zed@example.com"}`, list)); status != http.StatusNotFound {
		t.Fatalf("stopped job applied its upload: %d", status)
	}
	if jobs := mmgrOK(t, ts, "ListAddressListImportJobs", fmt.Sprintf(`{"AddressListId":%q}`, list))["ImportJobs"].([]any); len(jobs) != 3 {
		t.Fatalf("ListAddressListImportJobs: %d", len(jobs))
	}
}

func TestMailManagerStage1Limits(t *testing.T) {
	st := newMailManagerStore()
	st.mu.Lock()
	a, _ := st.add("archive", map[string]any{"ArchiveName": "a"}, map[string]any{}, time.Now())
	a.status = mmgrStatusActive
	st.mu.Unlock()
	big := &mmgrMessage{id: "big", raw: make([]byte, mmgrMaxMessageBytes+1)}
	if st.archive(a.id, big) {
		t.Fatal("archived a message over the size limit")
	}
	for i := 0; i < mmgrMaxMessages+5; i++ {
		st.archive(a.id, &mmgrMessage{id: fmt.Sprint(i), raw: []byte("x")})
	}
	if len(st.msgs) != mmgrMaxMessages || len(st.msgOrder) != mmgrMaxMessages || st.msgs["0"] != nil {
		t.Fatalf("eviction: %d %d", len(st.msgs), len(st.msgOrder))
	}
	if code, _ := st.upload("nope", "", []byte("x")); code != http.StatusForbidden {
		t.Fatalf("upload to a missing job: %d", code)
	}
}
