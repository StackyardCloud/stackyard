package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func issCall(t *testing.T, ts *httptest.Server, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/scan/sbom", []byte(body), map[string]string{"Content-Type": "application/json"}, "inspector-scan")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return resp.StatusCode, out
}

const issCycloneDX = `{"bomFormat":"CycloneDX","specVersion":"1.5","components":[
 {"bom-ref":"log4j","type":"library","name":"log4j-core","version":"2.14.1","purl":"pkg:maven/org.apache.logging.log4j/log4j-core@2.14.1"},
 {"bom-ref":"lodash","type":"library","name":"lodash","version":"4.17.21","purl":"pkg:npm/lodash@4.17.21"},
 {"type":"library","name":"requests","version":"2.28.0","components":[{"name":"x/net","version":"0.6.0","purl":"pkg:golang/golang.org/x/net@v0.6.0"}]}]}`

func TestInspectorScanStage1CycloneDX(t *testing.T) {
	_, ts := newInspectorTestServer(t)
	status, out := issCall(t, ts, `{"sbom":`+issCycloneDX+`,"outputFormat":"CYCLONE_DX_1_5"}`)
	if status != http.StatusOK {
		t.Fatalf("scan: %d %v", status, out)
	}
	bom := out["sbom"].(map[string]any)
	var ids []string
	for _, v := range bom["vulnerabilities"].([]any) {
		ids = append(ids, v.(map[string]any)["id"].(string))
	}
	// log4j 2.14.1 (< 2.15.0), requests 2.28.0 by name (< 2.31.0) and x/net
	// 0.6.0 (< 0.7.0) are affected; lodash 4.17.21 is the fixed release.
	if got := strings.Join(ids, ","); got != "CVE-2021-44228,CVE-2023-32681,CVE-2022-41723" {
		t.Fatalf("vulnerabilities: %s", got)
	}
	if bom["specVersion"] != "1.5" || len(bom["components"].([]any)) != 4 || !strings.Contains(fmt.Sprint(bom), "STACKYARD_SAMPLE_ADVISORIES") {
		t.Fatalf("report: %v", bom)
	}
}

func TestInspectorScanStage1SPDXAndErrors(t *testing.T) {
	_, ts := newInspectorTestServer(t)
	spdx := `{"spdxVersion":"SPDX-2.3","packages":[{"SPDXID":"SPDXRef-1","name":"lodash","versionInfo":"4.17.20","externalRefs":[{"referenceType":"purl","referenceLocator":"pkg:npm/lodash@4.17.20"}]}]}`
	status, out := issCall(t, ts, `{"sbom":`+spdx+`,"outputFormat":"INSPECTOR"}`)
	if status != http.StatusOK || !strings.Contains(fmt.Sprint(out), "CVE-2021-23337") {
		t.Fatalf("spdx scan: %d %v", status, out)
	}
	for name, body := range map[string]string{
		"empty sbom":      `{"sbom":{}}`,
		"old cyclonedx":   `{"sbom":{"bomFormat":"CycloneDX","specVersion":"1.2","components":[]}}`,
		"spdx 2.2":        `{"sbom":{"spdxVersion":"SPDX-2.2","packages":[]}}`,
		"nameless":        `{"sbom":{"bomFormat":"CycloneDX","specVersion":"1.5","components":[{"version":"1"}]}}`,
		"bad format":      `{"sbom":` + issCycloneDX + `,"outputFormat":"XML"}`,
		"unknown member":  `{"sbom":{},"extra":1}`,
		"not json object": `[1]`,
	} {
		status, out := issCall(t, ts, body)
		if status != http.StatusBadRequest || out["reason"] == nil {
			t.Fatalf("%s: %d %v", name, status, out)
		}
	}
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/scan/sbom", nil, nil, "inspector-scan")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /scan/sbom: %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestInspectorScanStage1VersionCompare(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{{"2.14.1", "2.15.0", -1}, {"2.15.0", "2.15.0", 0}, {"v0.6.0", "0.7.0", -1}, {"4.17.21", "4.17.3", 1}} {
		if got := issCompareVersions(c.a, c.b); got != c.want {
			t.Fatalf("%s vs %s: %d", c.a, c.b, got)
		}
	}
}
