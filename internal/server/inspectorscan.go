package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Amazon Inspector Scan (REST-JSON, POST /scan/sbom, SigV4 "inspector-scan").
// Vulnerabilities come from issSampleAdvisories, a small embedded sample list:
// Stackyard has no vulnerability database and makes no network calls.

const (
	issMaxBody       = 4 << 20
	issMaxComponents = 2000 // the AWS limit for one SBOM
)

type issError struct {
	status       int
	code, reason string
	msg          string
}

func issValidation(reason, format string, args ...any) *issError {
	return &issError{status: http.StatusBadRequest, code: "ValidationException", reason: reason, msg: fmt.Sprintf(format, args...)}
}

func (s *Server) handleInspectorScanRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	hint := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if hint != "inspector-scan" && !strings.HasPrefix(host, "inspector-scan.") && !strings.Contains(host, ".inspector-scan.") {
		return false
	}
	if r.ContentLength < 0 || r.ContentLength > issMaxBody {
		respondInspectorScanError(w, issValidation("OTHER", "request body must declare a Content-Length of at most 4 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "inspector-scan"); !ok {
		respondInspectorScanError(w, &issError{status: status, code: code, msg: msg})
		return true
	}
	if r.Method != http.MethodPost || rawRequestPath(r) != "/scan/sbom" {
		respondInspectorScanError(w, issValidation("UNKNOWN_OPERATION", "no Inspector Scan operation matches %s %s", r.Method, rawRequestPath(r)))
		return true
	}
	raw, err := readBodyBytes(r)
	var in struct {
		Sbom         json.RawMessage `json:"sbom"`
		OutputFormat string          `json:"outputFormat"`
	}
	if err == nil {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		err = dec.Decode(&in)
	}
	if err != nil {
		respondInspectorScanError(w, issValidation("CANNOT_PARSE", "request body must be a JSON object with sbom and outputFormat"))
		return true
	}
	out, ierr := issScan(in.Sbom, in.OutputFormat, time.Now().UTC())
	if ierr != nil {
		respondInspectorScanError(w, ierr)
		return true
	}
	body, _ := json.Marshal(map[string]any{"sbom": out})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondInspectorScanError(w http.ResponseWriter, e *issError) {
	body := map[string]any{"message": e.msg}
	if e.reason != "" {
		body["reason"] = e.reason
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

// issComponent is one SBOM component, normalised from CycloneDX or SPDX.
type issComponent struct {
	ref, kind, name, version, purl string
}

type issAdvisory struct {
	id, ecosystem, name, introduced, fixed, severity, description string
	score                                                         float64
}

// issSampleAdvisories is a fixed sample of real advisories, labelled as a
// sample in every report; it is not a vulnerability database.
var issSampleAdvisories = []issAdvisory{
	{id: "CVE-2021-44228", ecosystem: "maven", name: "org.apache.logging.log4j/log4j-core", introduced: "2.0", fixed: "2.15.0", severity: "critical", score: 10,
		description: "Apache Log4j2 JNDI features do not protect against attacker-controlled LDAP and other JNDI endpoints (Log4Shell)."},
	{id: "CVE-2022-42889", ecosystem: "maven", name: "org.apache.commons/commons-text", introduced: "1.5", fixed: "1.10.0", severity: "critical", score: 9.8,
		description: "Apache Commons Text variable interpolation can lead to arbitrary code execution (Text4Shell)."},
	{id: "CVE-2021-23337", ecosystem: "npm", name: "lodash", fixed: "4.17.21", severity: "high", score: 7.2,
		description: "Lodash versions prior to 4.17.21 are vulnerable to command injection via the template function."},
	{id: "CVE-2022-41723", ecosystem: "golang", name: "golang.org/x/net", fixed: "0.7.0", severity: "high", score: 7.5,
		description: "A maliciously crafted HTTP/2 stream could cause excessive CPU consumption in the HPACK decoder."},
	{id: "CVE-2023-32681", ecosystem: "pypi", name: "requests", introduced: "2.3.0", fixed: "2.31.0", severity: "medium", score: 6.1,
		description: "Requests leaks Proxy-Authorization headers to destination servers when redirected to an HTTPS endpoint."},
}

func issScan(raw json.RawMessage, format string, now time.Time) (map[string]any, *issError) {
	if format == "" {
		format = "CYCLONE_DX_1_5"
	}
	if format != "CYCLONE_DX_1_5" && format != "INSPECTOR" && format != "INSPECTOR_ALT" {
		return nil, issValidation("FIELD_VALIDATION_FAILED", "outputFormat must be one of CYCLONE_DX_1_5, INSPECTOR, INSPECTOR_ALT")
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil || doc == nil {
		return nil, issValidation("FIELD_VALIDATION_FAILED", "sbom must be a JSON object")
	}
	components, err := issComponents(doc)
	if err != nil {
		return nil, err
	}
	type hit struct {
		adv  issAdvisory
		comp issComponent
	}
	var hits []hit
	for _, c := range components {
		for _, adv := range issSampleAdvisories {
			if issAffected(adv, c) {
				hits = append(hits, hit{adv, c})
			}
		}
	}
	counts := map[string]int{}
	for _, h := range hits {
		counts[h.adv.severity]++
	}
	if format != "CYCLONE_DX_1_5" {
		// ponytail: INSPECTOR and INSPECTOR_ALT share one flat layout here.
		vulns := []any{}
		for _, h := range hits {
			vulns = append(vulns, map[string]any{
				"id": h.adv.id, "severity": strings.ToUpper(h.adv.severity), "score": h.adv.score, "description": h.adv.description,
				"fixed_version": h.adv.fixed, "source": "STACKYARD_SAMPLE_ADVISORIES",
				"affected_component": map[string]any{"name": h.comp.name, "version": h.comp.version, "purl": h.comp.purl},
				"references":         []string{"https://nvd.nist.gov/vuln/detail/" + h.adv.id},
			})
		}
		return map[string]any{
			"status": "SUCCEEDED", "advisory_source": "STACKYARD_SAMPLE_ADVISORIES",
			"summary": map[string]any{"total_components": len(components), "vulnerabilities": len(hits),
				"critical": counts["critical"], "high": counts["high"], "medium": counts["medium"], "low": counts["low"]},
			"vulnerabilities": vulns,
		}, nil
	}
	comps := []any{}
	for _, c := range components {
		item := map[string]any{"bom-ref": c.ref, "type": c.kind, "name": c.name}
		if c.version != "" {
			item["version"] = c.version
		}
		if c.purl != "" {
			item["purl"] = c.purl
		}
		comps = append(comps, item)
	}
	vulns := []any{}
	for i, h := range hits {
		vulns = append(vulns, map[string]any{
			"bom-ref":     fmt.Sprintf("vuln-%d", i+1),
			"id":          h.adv.id,
			"source":      map[string]any{"name": "STACKYARD_SAMPLE_ADVISORIES", "url": "https://nvd.nist.gov/vuln/detail/" + h.adv.id},
			"ratings":     []any{map[string]any{"source": map[string]any{"name": "NVD"}, "score": h.adv.score, "severity": h.adv.severity, "method": "CVSSv31"}},
			"description": h.adv.description,
			"advisories":  []any{map[string]any{"url": "https://nvd.nist.gov/vuln/detail/" + h.adv.id}},
			"affects":     []any{map[string]any{"ref": h.comp.ref}},
			"properties":  []any{map[string]any{"name": "amazon:inspector:sbom_scanner:fixed_version:" + h.comp.ref, "value": h.adv.fixed}},
		})
	}
	props := []any{}
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		props = append(props, map[string]any{"name": "amazon:inspector:sbom_scanner:" + sev + "_vulnerabilities", "value": strconv.Itoa(counts[sev])})
	}
	return map[string]any{
		"bomFormat": "CycloneDX", "specVersion": "1.5", "serialNumber": "urn:uuid:" + uuid.NewString(), "version": 1,
		"metadata": map[string]any{
			"timestamp":  now.Format(time.RFC3339),
			"tools":      map[string]any{"components": []any{map[string]any{"type": "application", "name": "Stackyard Inspector Scan", "version": "1.0"}}},
			"properties": props,
		},
		"components": comps, "vulnerabilities": vulns,
	}, nil
}

// issComponents reads components from a CycloneDX 1.4/1.5 or SPDX 2.3 document.
func issComponents(doc map[string]any) ([]issComponent, *issError) {
	str := func(m map[string]any, k string) string { v, _ := m[k].(string); return v }
	var out []issComponent
	add := func(c issComponent) *issError {
		if c.name == "" {
			return issValidation("FIELD_VALIDATION_FAILED", "every component needs a name")
		}
		if len(out) >= issMaxComponents {
			return issValidation("FIELD_VALIDATION_FAILED", "an SBOM can have at most %d components", issMaxComponents)
		}
		if c.ref == "" {
			c.ref = fmt.Sprintf("comp-%d", len(out)+1)
		}
		out = append(out, c)
		return nil
	}
	switch {
	case str(doc, "bomFormat") == "CycloneDX":
		if v := str(doc, "specVersion"); v != "1.4" && v != "1.5" {
			return nil, issValidation("UNSUPPORTED_SBOM_TYPE", "CycloneDX specVersion %q is not supported; use 1.4 or 1.5", v)
		}
		// Nested components are flattened; the component cap bounds the walk.
		queue, _ := doc["components"].([]any)
		for len(queue) > 0 {
			m, ok := queue[0].(map[string]any)
			queue = queue[1:]
			if !ok {
				return nil, issValidation("FIELD_VALIDATION_FAILED", "components must be objects")
			}
			kind := str(m, "type")
			if kind == "" {
				kind = "library"
			}
			if err := add(issComponent{ref: str(m, "bom-ref"), kind: kind, name: str(m, "name"), version: str(m, "version"), purl: str(m, "purl")}); err != nil {
				return nil, err
			}
			if nested, ok := m["components"].([]any); ok {
				queue = append(queue, nested...)
			}
		}
	case str(doc, "spdxVersion") != "":
		if str(doc, "spdxVersion") != "SPDX-2.3" {
			return nil, issValidation("UNSUPPORTED_SBOM_TYPE", "SPDX version %q is not supported; use SPDX-2.3", str(doc, "spdxVersion"))
		}
		packages, _ := doc["packages"].([]any)
		for _, p := range packages {
			m, ok := p.(map[string]any)
			if !ok {
				return nil, issValidation("FIELD_VALIDATION_FAILED", "packages must be objects")
			}
			c := issComponent{ref: str(m, "SPDXID"), kind: "library", name: str(m, "name"), version: str(m, "versionInfo")}
			refs, _ := m["externalRefs"].([]any)
			for _, r := range refs {
				if rm, ok := r.(map[string]any); ok && str(rm, "referenceType") == "purl" {
					c.purl = str(rm, "referenceLocator")
				}
			}
			if err := add(c); err != nil {
				return nil, err
			}
		}
	default:
		return nil, issValidation("UNSUPPORTED_SBOM_TYPE", "sbom must be a CycloneDX 1.4/1.5 or SPDX 2.3 JSON document")
	}
	return out, nil
}

// issAffected reports whether a component falls in an advisory's range,
// matching on purl type and name when a purl is given, else on name.
func issAffected(adv issAdvisory, c issComponent) bool {
	name, version := c.name, c.version
	if c.purl != "" {
		kind, pname, pversion, ok := issParsePurl(c.purl)
		if !ok || kind != adv.ecosystem {
			return false
		}
		name = pname
		if pversion != "" {
			version = pversion
		}
	} else {
		// Without a purl, a bare name matches the advisory's package name.
		if adv.name != name && !strings.HasSuffix(adv.name, "/"+name) {
			return false
		}
		name = adv.name
	}
	if name != adv.name || version == "" {
		return false
	}
	if adv.introduced != "" && issCompareVersions(version, adv.introduced) < 0 {
		return false
	}
	return issCompareVersions(version, adv.fixed) < 0
}

// issParsePurl splits pkg:type/namespace/name@version into its parts.
func issParsePurl(purl string) (kind, name, version string, ok bool) {
	rest, found := strings.CutPrefix(purl, "pkg:")
	if !found {
		return "", "", "", false
	}
	rest, _, _ = strings.Cut(rest, "?")
	rest, _, _ = strings.Cut(rest, "#")
	rest, version, _ = strings.Cut(rest, "@")
	kind, name, found = strings.Cut(rest, "/")
	if !found || name == "" {
		return "", "", "", false
	}
	if n, err := url.PathUnescape(name); err == nil {
		name = n
	}
	if v, err := url.PathUnescape(version); err == nil {
		version = v
	}
	return strings.ToLower(kind), name, version, true
}

// issCompareVersions compares dotted versions part by part: numerically
// when both parts are numbers, otherwise as strings.
func issCompareVersions(a, b string) int {
	split := func(v string) []string {
		return strings.FieldsFunc(strings.TrimPrefix(v, "v"), func(r rune) bool { return r == '.' || r == '-' || r == '+' })
	}
	pa, pb := split(a), split(b)
	for i := 0; i < max(len(pa), len(pb)); i++ {
		x, y := "0", "0"
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		nx, ex := strconv.ParseInt(x, 10, 64)
		ny, ey := strconv.ParseInt(y, 10, 64)
		switch {
		case ex == nil && ey == nil && nx != ny:
			if nx < ny {
				return -1
			}
			return 1
		case (ex != nil || ey != nil) && x != y:
			return strings.Compare(x, y)
		}
	}
	return 0
}
