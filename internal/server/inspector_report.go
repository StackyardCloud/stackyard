package server

import (
	"fmt"
	"html"
	"net/http"
	"strings"
)

// iscReportPath serves reports from GetAssessmentReport. The URL token is
// the capability, as with the presigned S3 URLs AWS returns.
const iscReportPath = "/_stackyard/inspector/reports/"

func (s *Server) handleInspectorReport(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if r.Method != http.MethodGet || !strings.HasPrefix(path, iscReportPath) {
		return false
	}
	token, format, _ := strings.Cut(strings.TrimPrefix(path, iscReportPath), ".")
	lines, ok := s.inspector.reportLines(token, format)
	if !ok {
		respondInspectorJSON(w, http.StatusNotFound, map[string]any{"__type": "NoSuchEntityException", "message": "report not found"})
		return true
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if format == "pdf" {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(iscPDF(lines))
		return true
	}
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>Amazon Inspector assessment report</title></head><body>")
	for i, line := range lines {
		tag := "p"
		if i == 0 {
			tag = "h1"
		}
		fmt.Fprintf(&b, "<%s>%s</%s>", tag, html.EscapeString(line), tag)
	}
	b.WriteString("</body></html>")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
	return true
}

// reportLines renders a report's text under the lock.
func (st *iscStore) reportLines(token, format string) ([]string, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	rep, ok := st.reports[token]
	run := st.runs[rep.runArn]
	if !ok || run == nil || rep.format != format {
		return nil, false
	}
	lines := []string{"Amazon Inspector assessment report", "Run: " + run.name, "Run ARN: " + run.arn, "Report type: " + rep.kind,
		fmt.Sprintf("Instances assessed: %d", len(run.agents)), fmt.Sprintf("Findings: %d", len(run.findings))}
	for _, arn := range run.findings {
		f := st.findings[arn]
		rule := iscRulesPackageByARN(f.packageArn).rule
		lines = append(lines, "["+rule.severity+"] "+fmt.Sprintf(rule.title, f.instance.ID))
	}
	return lines, true
}

// iscPDF builds a minimal one-page PDF listing lines (ASCII only).
func iscPDF(lines []string) []byte {
	var content strings.Builder
	content.WriteString("BT /F1 10 Tf 40 800 Td 14 TL\n")
	for i, line := range lines {
		if i >= 55 {
			content.WriteString("(...) '\n")
			break
		}
		clean := strings.Map(func(r rune) rune {
			if r < 32 || r > 126 {
				return '?'
			}
			return r
		}, line)
		clean = strings.NewReplacer(`\`, `\\`, "(", `\(`, ")", `\)`).Replace(clean)
		fmt.Fprintf(&content, "(%s) '\n", clean)
	}
	content.WriteString("ET")
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", content.Len(), content.String()),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var out strings.Builder
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return []byte(out.String())
}
