package server

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGlacierStage1VaultsAndArchives(t *testing.T) {
	_, ts := newGlacierTestServer(t)
	h, _ := glcWant(t, ts, http.StatusCreated, http.MethodPut, "/-/vaults/photos", nil, nil)
	if h.Get("Location") != "/"+glcAccountID+"/vaults/photos" {
		t.Fatalf("Location: %q", h.Get("Location"))
	}
	glcWant(t, ts, http.StatusCreated, http.MethodPut, "/-/vaults/photos", nil, nil) // idempotent
	if code := glcErrorCode(t, ts, http.MethodPut, "/-/vaults/bad%20name", nil, nil); code != "InvalidParameterValueException" {
		t.Fatalf("bad vault name: %s", code)
	}
	if code := glcErrorCode(t, ts, http.MethodGet, "/-/vaults/missing", nil, nil); code != "ResourceNotFoundException" {
		t.Fatalf("missing vault: %s", code)
	}

	data := []byte("hello glacier")
	hash := glcTreeHash(data)
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/photos/archives", data, map[string]string{"x-amz-sha256-tree-hash": strings.Repeat("0", 64)}); code != "InvalidParameterValueException" {
		t.Fatalf("checksum mismatch: %s", code)
	}
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/photos/archives", data, nil); code != "MissingParameterValueException" {
		t.Fatalf("missing checksum: %s", code)
	}
	h, _ = glcWant(t, ts, http.StatusCreated, http.MethodPost, "/-/vaults/photos/archives", data, map[string]string{"x-amz-sha256-tree-hash": hash, "x-amz-archive-description": "greeting"})
	archiveID := h.Get("x-amz-archive-id")
	if len(archiveID) != 138 || h.Get("x-amz-sha256-tree-hash") != hash {
		t.Fatalf("upload headers: %v", h)
	}

	// Counts come from the last inventory, so a fresh vault still reports 0.
	_, d := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/vaults/photos", nil, nil)
	if d["NumberOfArchives"] != float64(0) || d["LastInventoryDate"] != nil || d["VaultARN"] != glcVaultARN("photos") {
		t.Fatalf("describe before inventory: %v", d)
	}
	if code := glcErrorCode(t, ts, http.MethodDelete, "/-/vaults/photos", nil, nil); code != "InvalidParameterValueException" {
		t.Fatalf("delete non-empty vault: %s", code)
	}
	glcWant(t, ts, http.StatusCreated, http.MethodPut, "/-/vaults/other", nil, nil)
	_, list := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/vaults?limit=1", nil, nil)
	if len(list["VaultList"].([]any)) != 1 || list["Marker"] != glcVaultARN("photos") {
		t.Fatalf("paged list: %v", list)
	}
	_, list = glcWant(t, ts, http.StatusOK, http.MethodGet, fmt.Sprintf("/-/vaults?marker=%s", list["Marker"]), nil, nil)
	if len(list["VaultList"].([]any)) != 1 || list["Marker"] != nil {
		t.Fatalf("second page: %v", list)
	}

	glcWant(t, ts, http.StatusNoContent, http.MethodDelete, "/-/vaults/photos/archives/"+archiveID, nil, nil)
	if code := glcErrorCode(t, ts, http.MethodDelete, "/-/vaults/photos/archives/"+archiveID, nil, nil); code != "ResourceNotFoundException" {
		t.Fatalf("delete twice: %s", code)
	}
	glcWant(t, ts, http.StatusNoContent, http.MethodDelete, "/-/vaults/photos", nil, nil)
}

func TestGlacierStage1MultipartUpload(t *testing.T) {
	_, ts := newGlacierTestServer(t)
	glcWant(t, ts, http.StatusCreated, http.MethodPut, "/-/vaults/big", nil, nil)
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/big/multipart-uploads", nil, map[string]string{"x-amz-part-size": "3000000"}); code != "InvalidParameterValueException" {
		t.Fatalf("non power-of-two part size: %s", code)
	}
	h, _ := glcWant(t, ts, http.StatusCreated, http.MethodPost, "/-/vaults/big/multipart-uploads", nil, map[string]string{"x-amz-part-size": "1048576", "x-amz-archive-description": "two parts"})
	id := h.Get("x-amz-multipart-upload-id")
	base := "/-/vaults/big/multipart-uploads/" + id

	whole := bytes.Repeat([]byte("abcdefgh"), (1<<20+300)/8)
	first, second := whole[:1<<20], whole[1<<20:]
	part := func(start int, data []byte) map[string]string {
		return map[string]string{"Content-Range": fmt.Sprintf("bytes %d-%d/*", start, start+len(data)-1), "x-amz-sha256-tree-hash": glcTreeHash(data)}
	}
	if code := glcErrorCode(t, ts, http.MethodPut, base, second, part(5, second)); code != "InvalidParameterValueException" {
		t.Fatalf("misaligned part: %s", code)
	}
	glcWant(t, ts, http.StatusNoContent, http.MethodPut, base, second, part(1<<20, second))
	// Completing with a gap at the start is rejected.
	if code := glcErrorCode(t, ts, http.MethodPost, base, nil, map[string]string{"x-amz-archive-size": fmt.Sprint(len(second)), "x-amz-sha256-tree-hash": glcTreeHash(second)}); code != "InvalidParameterValueException" {
		t.Fatalf("gap: %s", code)
	}
	glcWant(t, ts, http.StatusNoContent, http.MethodPut, base, first, part(0, first))
	_, parts := glcWant(t, ts, http.StatusOK, http.MethodGet, base, nil, nil)
	if len(parts["Parts"].([]any)) != 2 || parts["PartSizeInBytes"] != float64(1<<20) {
		t.Fatalf("list parts: %v", parts)
	}
	_, ups := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/vaults/big/multipart-uploads", nil, nil)
	if len(ups["UploadsList"].([]any)) != 1 {
		t.Fatalf("list uploads: %v", ups)
	}
	if code := glcErrorCode(t, ts, http.MethodPost, base, nil, map[string]string{"x-amz-archive-size": fmt.Sprint(len(whole)), "x-amz-sha256-tree-hash": glcTreeHash(first)}); code != "InvalidParameterValueException" {
		t.Fatalf("wrong total hash: %s", code)
	}
	h, _ = glcWant(t, ts, http.StatusCreated, http.MethodPost, base, nil, map[string]string{"x-amz-archive-size": fmt.Sprint(len(whole)), "x-amz-sha256-tree-hash": glcTreeHash(whole)})
	if h.Get("x-amz-sha256-tree-hash") != glcTreeHash(whole) || h.Get("x-amz-archive-id") == "" {
		t.Fatalf("complete headers: %v", h)
	}
	if code := glcErrorCode(t, ts, http.MethodGet, base, nil, nil); code != "ResourceNotFoundException" {
		t.Fatalf("completed upload still listed: %s", code)
	}

	// Abort frees an upload.
	h, _ = glcWant(t, ts, http.StatusCreated, http.MethodPost, "/-/vaults/big/multipart-uploads", nil, map[string]string{"x-amz-part-size": "1048576"})
	glcWant(t, ts, http.StatusNoContent, http.MethodDelete, "/-/vaults/big/multipart-uploads/"+h.Get("x-amz-multipart-upload-id"), nil, nil)
}

func TestGlacierStage1Jobs(t *testing.T) {
	srv, ts := newGlacierTestServer(t)
	glcWant(t, ts, http.StatusCreated, http.MethodPut, "/-/vaults/jobs", nil, nil)
	data := bytes.Repeat([]byte("0123456789"), 300000) // ~2.9 MiB
	h, _ := glcWant(t, ts, http.StatusCreated, http.MethodPost, "/-/vaults/jobs/archives", data, map[string]string{"x-amz-sha256-tree-hash": glcTreeHash(data), "x-amz-archive-description": "digits"})
	archiveID := h.Get("x-amz-archive-id")

	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/jobs/jobs", []byte(`{"Type":"archive-retrieval"}`), nil); code != "MissingParameterValueException" {
		t.Fatalf("no archive id: %s", code)
	}
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/jobs/jobs", []byte(`{"Type":"archive-retrieval","ArchiveId":"`+archiveID+`","RetrievalByteRange":"5-100"}`), nil); code != "InvalidParameterValueException" {
		t.Fatalf("unaligned range: %s", code)
	}
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/jobs/jobs", []byte(`{"Type":"archive-retrieval","ArchiveId":"`+archiveID+`","Bogus":1}`), nil); code != "InvalidParameterValueException" {
		t.Fatalf("unknown member: %s", code)
	}
	h, _ = glcWant(t, ts, http.StatusAccepted, http.MethodPost, "/-/vaults/jobs/jobs", []byte(`{"Type":"archive-retrieval","ArchiveId":"`+archiveID+`","Tier":"Expedited"}`), nil)
	jobID := h.Get("x-amz-job-id")
	jobPath := "/-/vaults/jobs/jobs/" + jobID

	// The first read sees the job running; output isn't ready yet.
	_, j := glcWant(t, ts, http.StatusOK, http.MethodGet, jobPath, nil, nil)
	if j["StatusCode"] != "InProgress" || j["Completed"] != false {
		t.Fatalf("first read: %v", j)
	}
	_, j = glcWant(t, ts, http.StatusOK, http.MethodGet, jobPath, nil, nil)
	if j["StatusCode"] != "Succeeded" || j["ArchiveSHA256TreeHash"] != glcTreeHash(data) || j["RetrievalByteRange"] != fmt.Sprintf("0-%d", len(data)-1) {
		t.Fatalf("second read: %v", j)
	}
	status, oh, body := glcCall(t, ts, http.MethodGet, jobPath+"/output", nil, nil)
	if status != http.StatusOK || !bytes.Equal(body, data) || oh.Get("Content-Type") != "application/octet-stream" || oh.Get("x-amz-sha256-tree-hash") != glcTreeHash(data) || oh.Get("x-amz-archive-description") != "digits" {
		t.Fatalf("output: %d %v (%d bytes)", status, oh, len(body))
	}
	status, oh, body = glcCall(t, ts, http.MethodGet, jobPath+"/output", nil, map[string]string{"Range": "bytes=0-1048575"})
	if status != http.StatusPartialContent || !bytes.Equal(body, data[:1<<20]) || oh.Get("Content-Range") != fmt.Sprintf("bytes 0-1048575/%d", len(data)) || oh.Get("x-amz-sha256-tree-hash") != glcTreeHash(data[:1<<20]) {
		t.Fatalf("ranged output: %d %v", status, oh)
	}
	if code := glcErrorCode(t, ts, http.MethodGet, jobPath+"/output", nil, map[string]string{"Range": "bytes=99999999-"}); code != "InvalidParameterValueException" {
		t.Fatalf("bad range: %s", code)
	}

	// Inventory jobs snapshot the vault counts and render JSON or CSV.
	h, _ = glcWant(t, ts, http.StatusAccepted, http.MethodPost, "/-/vaults/jobs/jobs", []byte(`{"Type":"inventory-retrieval","Format":"CSV"}`), nil)
	invPath := "/-/vaults/jobs/jobs/" + h.Get("x-amz-job-id")
	if code := glcErrorCode(t, ts, http.MethodGet, invPath+"/output", nil, nil); code != "InvalidParameterValueException" {
		t.Fatalf("output before completion: %s", code)
	}
	status, oh, body = glcCall(t, ts, http.MethodGet, invPath+"/output", nil, nil)
	if status != http.StatusOK || oh.Get("Content-Type") != "text/csv" || !strings.Contains(string(body), archiveID) {
		t.Fatalf("inventory output: %d %v %s", status, oh, body)
	}
	_, d := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/vaults/jobs", nil, nil)
	if d["NumberOfArchives"] != float64(1) || d["SizeInBytes"] != float64(len(data)) || d["LastInventoryDate"] == nil {
		t.Fatalf("describe after inventory: %v", d)
	}
	_, lj := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/vaults/jobs/jobs?statuscode=Succeeded&limit=1", nil, nil)
	if len(lj["JobList"].([]any)) != 1 || lj["Marker"] == nil {
		t.Fatalf("list jobs: %v", lj)
	}

	// A BytesPerHour retrieval policy refuses a bigger retrieval.
	glcWant(t, ts, http.StatusNoContent, http.MethodPut, "/-/policies/data-retrieval", []byte(`{"Policy":{"Rules":[{"Strategy":"BytesPerHour","BytesPerHour":1024}]}}`), nil)
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/jobs/jobs", []byte(`{"Type":"archive-retrieval","ArchiveId":"`+archiveID+`"}`), nil); code != "PolicyEnforcedException" {
		t.Fatalf("policy: %s", code)
	}
	if code := glcErrorCode(t, ts, http.MethodPut, "/-/policies/data-retrieval", []byte(`{"Policy":{"Rules":[{"Strategy":"None","BytesPerHour":5}]}}`), nil); code != "InvalidParameterValueException" {
		t.Fatalf("BytesPerHour with None: %s", code)
	}
	_ = srv
}

func TestGlacierStage1PoliciesLockTagsCapacity(t *testing.T) {
	srv, ts := newGlacierTestServer(t)
	glcWant(t, ts, http.StatusCreated, http.MethodPut, "/-/vaults/v", nil, nil)
	policy := []byte(`{"Policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)

	if code := glcErrorCode(t, ts, http.MethodGet, "/-/vaults/v/access-policy", nil, nil); code != "ResourceNotFoundException" {
		t.Fatalf("no access policy: %s", code)
	}
	if code := glcErrorCode(t, ts, http.MethodPut, "/-/vaults/v/access-policy", []byte(`{"Policy":"not json"}`), nil); code != "InvalidParameterValueException" {
		t.Fatalf("bad policy: %s", code)
	}
	glcWant(t, ts, http.StatusNoContent, http.MethodPut, "/-/vaults/v/access-policy", policy, nil)
	if _, p := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/vaults/v/access-policy", nil, nil); !strings.Contains(fmt.Sprint(p["Policy"]), "Statement") {
		t.Fatalf("access policy: %v", p)
	}
	glcWant(t, ts, http.StatusNoContent, http.MethodDelete, "/-/vaults/v/access-policy", nil, nil)

	// Notifications need an existing SNS topic.
	if code := glcErrorCode(t, ts, http.MethodPut, "/-/vaults/v/notification-configuration", []byte(`{"SNSTopic":"arn:aws:sns:us-east-1:123456789012:nope","Events":["ArchiveRetrievalCompleted"]}`), nil); code != "InvalidParameterValueException" {
		t.Fatalf("missing topic: %s", code)
	}
	topic, err := srv.sns.CreateTopic("glacier-events", nil)
	if err != nil {
		t.Fatal(err)
	}
	glcWant(t, ts, http.StatusNoContent, http.MethodPut, "/-/vaults/v/notification-configuration", []byte(`{"SNSTopic":"`+topic.ARN+`","Events":["ArchiveRetrievalCompleted"]}`), nil)
	if _, n := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/vaults/v/notification-configuration", nil, nil); n["SNSTopic"] != topic.ARN {
		t.Fatalf("notifications: %v", n)
	}

	// Vault lock: InProgress for 24 hours, then Locked once completed.
	h, _ := glcWant(t, ts, http.StatusCreated, http.MethodPost, "/-/vaults/v/lock-policy", policy, nil)
	lockID := h.Get("x-amz-lock-id")
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/v/lock-policy", policy, nil); code != "InvalidParameterValueException" {
		t.Fatalf("second initiate: %s", code)
	}
	if _, l := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/vaults/v/lock-policy", nil, nil); l["State"] != "InProgress" || l["ExpirationDate"] == nil {
		t.Fatalf("lock in progress: %v", l)
	}
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/v/lock-policy/wrong", nil, nil); code != "InvalidParameterValueException" {
		t.Fatalf("wrong lock id: %s", code)
	}
	glcWant(t, ts, http.StatusNoContent, http.MethodPost, "/-/vaults/v/lock-policy/"+lockID, nil, nil)
	if code := glcErrorCode(t, ts, http.MethodDelete, "/-/vaults/v/lock-policy", nil, nil); code != "InvalidParameterValueException" {
		t.Fatalf("abort a completed lock: %s", code)
	}
	// An uncompleted lock lapses after its window.
	glcWant(t, ts, http.StatusCreated, http.MethodPut, "/-/vaults/lapse", nil, nil)
	glcWant(t, ts, http.StatusCreated, http.MethodPost, "/-/vaults/lapse/lock-policy", policy, nil)
	saved := glcNow
	glcNow = func() time.Time { return saved().Add(25 * time.Hour) }
	if code := glcErrorCode(t, ts, http.MethodGet, "/-/vaults/lapse/lock-policy", nil, nil); code != "ResourceNotFoundException" {
		glcNow = saved
		t.Fatalf("lapsed lock: %s", code)
	}
	glcNow = saved

	// Tags.
	glcWant(t, ts, http.StatusNoContent, http.MethodPost, "/-/vaults/v/tags?operation=add", []byte(`{"Tags":{"team":"data","env":"dev"}}`), nil)
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/vaults/v/tags?operation=add", []byte(`{"Tags":{"aws:owner":"x"}}`), nil); code != "InvalidParameterValueException" {
		t.Fatalf("aws: tag: %s", code)
	}
	glcWant(t, ts, http.StatusNoContent, http.MethodPost, "/-/vaults/v/tags?operation=remove", []byte(`{"TagKeys":["env"]}`), nil)
	if _, tags := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/vaults/v/tags", nil, nil); fmt.Sprint(tags["Tags"]) != "map[team:data]" {
		t.Fatalf("tags: %v", tags)
	}

	// Provisioned capacity is limited to two units.
	for i := 0; i < glcMaxCapacityUnits; i++ {
		glcWant(t, ts, http.StatusCreated, http.MethodPost, "/-/provisioned-capacity", nil, nil)
	}
	if code := glcErrorCode(t, ts, http.MethodPost, "/-/provisioned-capacity", nil, nil); code != "LimitExceededException" {
		t.Fatalf("capacity limit: %s", code)
	}
	if _, c := glcWant(t, ts, http.StatusOK, http.MethodGet, "/-/provisioned-capacity", nil, nil); len(c["ProvisionedCapacityList"].([]any)) != glcMaxCapacityUnits {
		t.Fatalf("capacity list: %v", c)
	}
}

func TestGlacierStage1StorageBudget(t *testing.T) {
	st := newGlacierStore()
	if err := st.createVault("v"); err != nil {
		t.Fatal(err.msg)
	}
	st.bytes = glcMaxStoreBytes - 4
	data := []byte("12345")
	if _, err := st.uploadArchive("v", "", glcTreeHash(data), data); err == nil || err.code != "LimitExceededException" {
		t.Fatalf("over budget: %v", err)
	}
	u, _ := st.initiateUpload("v", "", glcMinPartSize)
	if err := st.uploadPart("v", u.id, glcTreeHash(data), 0, 4, data); err == nil || err.code != "LimitExceededException" {
		t.Fatalf("part over budget: %v", err)
	}
}

// Job outputs never outgrow the store budget: deleting an archive drops the
// bytes its retrieval jobs share, and inventories count until they're gone.
func TestGlacierStage1JobMemoryIsBounded(t *testing.T) {
	st := newGlacierStore()
	if err := st.createVault("v"); err != nil {
		t.Fatal(err.msg)
	}
	data := []byte("job bytes")
	a, err := st.uploadArchive("v", "", glcTreeHash(data), data)
	if err != nil {
		t.Fatal(err.msg)
	}
	j, err := st.initiateJob("v", &glcJobParams{Type: "archive-retrieval", ArchiveID: a.id})
	if err != nil {
		t.Fatal(err.msg)
	}
	st.describeJob("v", j.id)
	st.describeJob("v", j.id)
	if err := st.deleteArchive("v", a.id); err != nil {
		t.Fatal(err.msg)
	}
	if _, err := st.jobOutput("v", j.id); err == nil || err.code != "ResourceNotFoundException" || j.data != nil {
		t.Fatalf("output after archive delete: %v (data %d bytes)", err, len(j.data))
	}

	inv, _ := st.initiateJob("v", &glcJobParams{Type: "inventory-retrieval"})
	st.describeJob("v", inv.id)
	st.describeJob("v", inv.id)
	if inv.held == 0 || st.bytes != inv.held {
		t.Fatalf("inventory not counted: held %d, store %d", inv.held, st.bytes)
	}
	if err := st.deleteVault("v"); err != nil || st.bytes != 0 {
		t.Fatalf("vault delete should release job bytes: %v, %d left", err, st.bytes)
	}

	// An inventory that doesn't fit the budget fails instead of growing memory.
	_ = st.createVault("w")
	st.bytes = glcMaxStoreBytes
	full, _ := st.initiateJob("w", &glcJobParams{Type: "inventory-retrieval"})
	st.describeJob("w", full.id)
	if d, _ := st.describeJob("w", full.id); d["StatusCode"] != "Failed" {
		t.Fatalf("over-budget inventory: %v", d)
	}
}
