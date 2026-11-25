package server

import (
	"bytes"
	"crypto/subtle"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Mail Manager archives, searches, exports and address list imports.
//
// Messages reach an archive when SES (SESv2 or Pinpoint Email) sends through
// a configuration set whose archiving options name it. Searches and exports
// snapshot the matching messages when they start and settle on the next read
// (RUNNING/PROCESSING, then COMPLETED); until then they can be stopped.
// Import jobs take a CSV or JSON upload at their pre-signed URL and apply it
// when the started job settles.

const (
	mmgrMaxMessages       = 5000
	mmgrMaxMessageBytes   = 1 << 20  // larger messages aren't archived
	mmgrMaxArchiveBytes   = 32 << 20 // all archives together; the oldest messages are evicted
	mmgrMaxUpload         = 1 << 20  // per import job
	mmgrMaxUploadTotal    = 16 << 20 // pending uploads together
	mmgrMaxExportMessages = 1000
	mmgrLinkPrefix        = "/_stackyard/mailmanager/"
	mmgrMessagePath       = mmgrLinkPrefix + "archived-messages/"
	mmgrImportPath        = mmgrLinkPrefix + "address-list-import/"
)

type mmgrMessage struct {
	id, archiveID            string
	received                 time.Time
	from, subject, messageID string
	to, cc, envelopeTo       []string
	text, html               string
	raw                      []byte
	configSet, sourceARN     string
}

var mmgrArchiveHandlers = map[string]mmgrHandler{
	"CreateAddressListImportJob": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		return s.mailmanager.createImportJob(q)
	},
	"StartAddressListImportJob": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		return nil, s.mailmanager.transition("importJob", mmgrStr(q.in, "JobId"), "CREATED", "PROCESSING")
	},
	"StopAddressListImportJob": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		return nil, s.mailmanager.transition("importJob", mmgrStr(q.in, "JobId"), "PROCESSING", "STOPPED")
	},
	"StartArchiveSearch": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		r, err := s.mailmanager.startSnapshot("search", q.in, mmgrInt(q.in, "MaxResults", 0))
		if err != nil {
			return nil, err
		}
		return map[string]any{"SearchId": r.id}, nil
	},
	"StopArchiveSearch": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		return nil, s.mailmanager.transition("search", mmgrStr(q.in, "SearchId"), "RUNNING", "CANCELLED")
	},
	"GetArchiveSearchResults": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		return s.mailmanager.searchResults(mmgrStr(q.in, "SearchId"))
	},
	"StartArchiveExport": (*Server).mmgrStartExport,
	"StopArchiveExport": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		return nil, s.mailmanager.transition("export", mmgrStr(q.in, "ExportId"), "PROCESSING", "CANCELLED")
	},
	"GetArchiveMessage": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		st := s.mailmanager
		st.mu.Lock()
		defer st.mu.Unlock()
		m, err := st.message(mmgrStr(q.in, "ArchivedMessageId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"MessageDownloadLink": q.base + mmgrMessagePath + m.id,
			"Metadata":            mmgrMetadata(m),
			"Envelope":            mmgrEnvelope(m),
		}, nil
	},
	"GetArchiveMessageContent": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		st := s.mailmanager
		st.mu.Lock()
		defer st.mu.Unlock()
		m, err := st.message(mmgrStr(q.in, "ArchivedMessageId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"Body": map[string]any{"Text": m.text, "Html": m.html, "MessageMalformed": false}}, nil
	},
}

func (st *mmgrStore) message(id string) (*mmgrMessage, *mmgrErr) {
	if m := st.msgs[id]; m != nil {
		return m, nil
	}
	return nil, mmgrError("ResourceNotFoundException", "Archived message %s was not found.", id)
}

func mmgrMetadata(m *mmgrMessage) map[string]any {
	return map[string]any{
		"Timestamp": mmgrEpoch(m.received), "SendingMethod": "SES", "SourceIdentity": m.from,
		"ConfigurationSet": m.configSet, "SourceArn": m.sourceARN,
	}
}

func mmgrEnvelope(m *mmgrMessage) map[string]any {
	return map[string]any{"From": m.from, "To": append([]any(nil), mmgrAnyStrings(m.envelopeTo)...)}
}

func mmgrAnyStrings(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}

// ---- archiving ----

// archive stores m in the ACTIVE archive ref (id or ARN). Messages for a
// missing or deleted archive, or over the size limit, are dropped.
func (st *mmgrStore) archive(ref string, m *mmgrMessage) bool {
	if len(m.raw) > mmgrMaxMessageBytes {
		return false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	a := st.resolve("archive", ref)
	if a == nil || a.status != mmgrStatusActive {
		return false
	}
	m.archiveID = a.id
	st.msgs[m.id] = m
	st.msgOrder = append(st.msgOrder, m)
	st.msgBytes += len(m.raw)
	for len(st.msgOrder) > mmgrMaxMessages || st.msgBytes > mmgrMaxArchiveBytes {
		old := st.msgOrder[0]
		st.msgOrder = st.msgOrder[1:]
		st.msgBytes -= len(old.raw)
		delete(st.msgs, old.id)
	}
	return true
}

func (st *mmgrStore) dropMessages(archiveID string) {
	kept := make([]*mmgrMessage, 0, len(st.msgOrder))
	for _, m := range st.msgOrder {
		if m.archiveID == archiveID {
			st.msgBytes -= len(m.raw)
			delete(st.msgs, m.id)
			continue
		}
		kept = append(kept, m)
	}
	st.msgOrder = kept
}

// mmgrArchiveSESSend archives a message SES just sent when its configuration
// set has archiving options. payload is the SendEmail request (SESv2 and
// Pinpoint Email share the shape).
func (s *Server) mmgrArchiveSESSend(payload map[string]any, sourceARN string) {
	configSet := mmgrStr(payload, "ConfigurationSetName")
	if configSet == "" {
		return
	}
	cfg, ok := s.sesv2.GetConfigurationSet(configSet)
	if !ok || cfg.ArchiveArn == "" {
		return
	}
	m := s.mmgrBuildMessage(payload)
	if m == nil {
		return
	}
	m.configSet, m.sourceARN = configSet, sourceARN
	s.mailmanager.archive(cfg.ArchiveArn, m)
}

// mmgrBuildMessage renders a SendEmail request (Simple, Raw or Template
// content) into a stored message with an RFC 822 copy, or nil when the
// content can't be read.
func (s *Server) mmgrBuildMessage(payload map[string]any) *mmgrMessage {
	dest := mmgrMap(payload, "Destination")
	m := &mmgrMessage{
		id: uuid.NewString(), received: time.Now().UTC(), from: mmgrStr(payload, "FromEmailAddress"),
		to: mmgrStrings(dest["ToAddresses"]), cc: mmgrStrings(dest["CcAddresses"]),
	}
	m.messageID = "<" + m.id + "@email.amazonses.com>"
	content := mmgrMap(payload, "Content")
	switch {
	case content["Simple"] != nil:
		simple := mmgrMap(content, "Simple")
		body := mmgrMap(simple, "Body")
		m.subject = mmgrStr(mmgrMap(simple, "Subject"), "Data")
		m.text, m.html = mmgrStr(mmgrMap(body, "Text"), "Data"), mmgrStr(mmgrMap(body, "Html"), "Data")
	case content["Template"] != nil:
		tpl := mmgrMap(content, "Template")
		name := mmgrStr(tpl, "TemplateName")
		if name == "" {
			arn := mmgrStr(tpl, "TemplateArn")
			name = arn[strings.LastIndex(arn, "/")+1:]
		}
		t, ok := s.sesv2.GetEmailTemplate(name)
		if !ok {
			return nil
		}
		data := map[string]any{}
		_ = json.Unmarshal([]byte(mmgrStr(tpl, "TemplateData")), &data)
		m.subject, m.text, m.html = mmgrRender(t.Subject, data), mmgrRender(t.Text, data), mmgrRender(t.HTML, data)
	case content["Raw"] != nil:
		raw, err := base64.StdEncoding.DecodeString(mmgrStr(mmgrMap(content, "Raw"), "Data"))
		if err != nil || len(raw) > mmgrMaxMessageBytes {
			return nil
		}
		parsed, err := mail.ReadMessage(bytes.NewReader(raw))
		if err != nil {
			return nil
		}
		body, _ := io.ReadAll(io.LimitReader(parsed.Body, mmgrMaxMessageBytes))
		h := parsed.Header
		m.subject = h.Get("Subject")
		if dec, err := new(mime.WordDecoder).DecodeHeader(m.subject); err == nil {
			m.subject = dec
		}
		if m.from == "" {
			m.from = mmgrAddrOnly(h.Get("From"))
		}
		if len(m.to) == 0 {
			m.to, m.cc = mmgrAddrList(h.Get("To")), mmgrAddrList(h.Get("Cc"))
		}
		if id := h.Get("Message-Id"); id != "" {
			m.messageID = id
		}
		if strings.HasPrefix(strings.ToLower(h.Get("Content-Type")), "text/html") {
			m.html = string(body)
		} else {
			m.text = string(body)
		}
		m.raw = raw
	default:
		return nil
	}
	m.envelopeTo = append(append(append([]string(nil), m.to...), m.cc...), mmgrStrings(dest["BccAddresses"])...)
	if m.raw == nil {
		m.raw = mmgrRFC822(m)
	}
	return m
}

func mmgrRender(tpl string, data map[string]any) string {
	for k, v := range data {
		tpl = strings.ReplaceAll(tpl, "{{"+k+"}}", fmt.Sprint(v))
	}
	return tpl
}

func mmgrAddrOnly(v string) string {
	if a, err := mail.ParseAddress(v); err == nil {
		return a.Address
	}
	return strings.TrimSpace(v)
}

func mmgrAddrList(v string) []string {
	list, err := mail.ParseAddressList(v)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, a := range list {
		out = append(out, a.Address)
	}
	return out
}

// mmgrHeader drops line breaks so caller values can't add headers.
func mmgrHeader(v string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(v)
}

// mmgrRFC822 writes a simple MIME message for m.
func mmgrRFC822(m *mmgrMessage) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", mmgrHeader(m.from))
	if len(m.to) > 0 {
		fmt.Fprintf(&b, "To: %s\r\n", mmgrHeader(strings.Join(m.to, ", ")))
	}
	if len(m.cc) > 0 {
		fmt.Fprintf(&b, "Cc: %s\r\n", mmgrHeader(strings.Join(m.cc, ", ")))
	}
	fmt.Fprintf(&b, "Subject: %s\r\nDate: %s\r\nMessage-ID: %s\r\nMIME-Version: 1.0\r\n",
		mime.QEncoding.Encode("utf-8", mmgrHeader(m.subject)), m.received.Format(time.RFC1123Z), m.messageID)
	switch {
	case m.text != "" && m.html != "":
		boundary := "stackyard-" + strings.ReplaceAll(m.id, "-", "")
		fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)
		fmt.Fprintf(&b, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", boundary, m.text)
		fmt.Fprintf(&b, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n--%s--\r\n", boundary, m.html, boundary)
	case m.html != "":
		fmt.Fprintf(&b, "Content-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n", m.html)
	default:
		fmt.Fprintf(&b, "Content-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", m.text)
	}
	return b.Bytes()
}

// ---- searches and exports ----

// matching returns the archive's messages in [from, to) that pass filters.
func (st *mmgrStore) matching(archiveID string, in map[string]any, limit int) ([]string, *mmgrErr) {
	from, okFrom := mmgrTime(in["FromTimestamp"])
	to, okTo := mmgrTime(in["ToTimestamp"])
	if !okFrom || !okTo || !from.Before(to) {
		return nil, mmgrInvalid("FromTimestamp must be before ToTimestamp.")
	}
	filters := mmgrMap(in, "Filters")
	var ids []string
	for _, m := range st.msgOrder {
		if len(ids) >= limit {
			break
		}
		if m.archiveID != archiveID || m.received.Before(from) || !m.received.Before(to) {
			continue
		}
		if mmgrAll(m, mmgrList(filters, "Include")) && !mmgrAny(m, mmgrList(filters, "Unless")) {
			ids = append(ids, m.id)
		}
	}
	return ids, nil
}

func mmgrAll(m *mmgrMessage, conds []any) bool {
	for _, c := range conds {
		if !mmgrCondition(m, c) {
			return false
		}
	}
	return true
}

func mmgrAny(m *mmgrMessage, conds []any) bool {
	for _, c := range conds {
		if mmgrCondition(m, c) {
			return true
		}
	}
	return false
}

func mmgrCondition(m *mmgrMessage, v any) bool {
	c, _ := v.(map[string]any)
	if se := mmgrMap(c, "StringExpression"); se != nil {
		var subject string
		switch mmgrStr(mmgrMap(se, "Evaluate"), "Attribute") {
		case "TO":
			subject = strings.Join(m.to, ",")
		case "CC":
			subject = strings.Join(m.cc, ",")
		case "FROM", "ENVELOPE_FROM":
			subject = m.from
		case "SUBJECT":
			subject = m.subject
		case "ENVELOPE_TO":
			subject = strings.Join(m.envelopeTo, ",")
		}
		subject = strings.ToLower(subject)
		for _, want := range mmgrStrings(se["Values"]) {
			if strings.Contains(subject, strings.ToLower(want)) {
				return true
			}
		}
		return false
	}
	if be := mmgrMap(c, "BooleanExpression"); be != nil {
		// HAS_ATTACHMENTS: SES-archived messages here carry no attachments.
		return mmgrStr(be, "Operator") == "IS_FALSE"
	}
	return true
}

// startSnapshot starts a search or export over an ACTIVE archive.
func (st *mmgrStore) startSnapshot(kind string, in map[string]any, limit int) (*mmgrRes, *mmgrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.get("archive", mmgrStr(in, "ArchiveId"))
	if err != nil {
		return nil, err
	}
	if a.status != mmgrStatusActive {
		return nil, mmgrError("ConflictException", "Archive %s is %s.", a.id, a.status)
	}
	ids, err := st.matching(a.id, in, limit)
	if err != nil {
		return nil, err
	}
	fields := mmgrPick(in, "ArchiveId", "Filters", "FromTimestamp", "ToTimestamp", "MaxResults", "ExportDestinationConfiguration")
	r, err := st.add(kind, map[string]any{}, fields, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.parent, r.rows, r.status = a.id, ids, "RUNNING"
	if kind == "export" {
		r.status = "PROCESSING"
	}
	return r, nil
}

func (st *mmgrStore) searchResults(id string) (map[string]any, *mmgrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get("search", id)
	if err != nil {
		return nil, err
	}
	st.settle(r, time.Now().UTC())
	if r.status != "COMPLETED" {
		return nil, mmgrError("ConflictException", "Search %s is %s.", r.id, r.status)
	}
	rows := make([]any, 0, len(r.rows))
	for _, mid := range r.rows {
		if m := st.msgs[mid]; m != nil {
			rows = append(rows, map[string]any{
				"ArchivedMessageId": m.id, "ReceivedTimestamp": mmgrEpoch(m.received), "Date": m.received.Format(time.RFC1123Z),
				"To": strings.Join(m.to, ", "), "From": m.from, "Cc": strings.Join(m.cc, ", "), "Subject": m.subject,
				"MessageId": m.messageID, "HasAttachments": false, "Envelope": mmgrEnvelope(m), "SourceArn": m.sourceARN,
			})
		}
	}
	return map[string]any{"Rows": rows}, nil
}

// mmgrStartExport snapshots the matching messages and writes them to
// Stackyard's S3 as <prefix>/<export id>/<message id>.eml (plus
// .metadata.json with IncludeMetadata).
func (s *Server) mmgrStartExport(q mmgrReq) (map[string]any, *mmgrErr) {
	dest := mmgrStr(mmgrMap(mmgrMap(q.in, "ExportDestinationConfiguration"), "S3"), "S3Location")
	if dest == "" {
		return nil, mmgrInvalid("ExportDestinationConfiguration.S3.S3Location is required.")
	}
	failure := s.pszOutputProblem(dest)
	limit := mmgrInt(q.in, "MaxResults", mmgrMaxExportMessages)
	st := s.mailmanager
	r, err := st.startSnapshot("export", q.in, min(limit, mmgrMaxExportMessages))
	if err != nil {
		return nil, err
	}
	type file struct {
		name string
		body []byte
	}
	var files []file
	st.mu.Lock()
	r.errMsg = failure
	if failure == "" {
		for _, id := range r.rows {
			m := st.msgs[id]
			if m == nil {
				continue
			}
			files = append(files, file{m.id + ".eml", m.raw})
			if v, _ := q.in["IncludeMetadata"].(bool); v {
				meta, _ := json.Marshal(map[string]any{"Metadata": mmgrMetadata(m), "Envelope": mmgrEnvelope(m)})
				files = append(files, file{m.id + ".metadata.json", meta})
			}
		}
	}
	exportID := r.id
	st.mu.Unlock()
	// S3 is written after the Mail Manager lock is released.
	for _, f := range files {
		s.pszWriteS3(strings.TrimSuffix(dest, "/")+"/"+exportID, f.name, f.body)
	}
	return map[string]any{"ExportId": exportID}, nil
}

// transition moves a job, search or export from one state to another.
func (st *mmgrStore) transition(kind, id, from, to string) *mmgrErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get(kind, id)
	if err != nil {
		return err
	}
	if r.status != from {
		code := "ValidationException"
		if kind == "importJob" {
			code = "ConflictException"
		}
		return mmgrError(code, "%s is %s, not %s.", r.id, r.status, from)
	}
	now := time.Now().UTC()
	r.status = to
	if to == "PROCESSING" {
		r.started = now
	} else {
		r.finished = now
		st.uploadBytes -= len(r.upload)
		r.upload = nil
	}
	return nil
}

// settle advances a resource whose state changes on its own.
func (st *mmgrStore) settle(r *mmgrRes, now time.Time) {
	switch {
	case r.kind.id == "ingressPoint" && r.status == mmgrStatusProvsn:
		r.status = mmgrStatusActive
	case r.kind.id == "search" && r.status == "RUNNING":
		r.status, r.finished = "COMPLETED", now
	case r.kind.id == "export" && r.status == "PROCESSING":
		r.status, r.finished = "COMPLETED", now
		if r.errMsg != "" {
			r.status = "FAILED"
		}
	case r.kind.id == "importJob" && r.status == "PROCESSING":
		st.runImport(r)
		r.finished = now
	}
}

func (st *mmgrStore) viewJob(r *mmgrRes, out map[string]any) {
	switch r.kind.id {
	case "importJob":
		out["Status"], out["AddressListId"] = r.status, r.parent
		out["ImportedItemsCount"], out["FailedItemsCount"] = r.imported, r.failed
		out["StartTimestamp"], out["CompletedTimestamp"] = mmgrEpoch(r.started), mmgrEpoch(r.finished)
		if r.errMsg != "" {
			out["Error"] = r.errMsg
		}
	case "export", "search":
		out["ArchiveId"] = r.parent
		status := map[string]any{"SubmissionTimestamp": mmgrEpoch(r.created), "CompletionTimestamp": mmgrEpoch(r.finished), "State": r.status}
		if r.errMsg != "" && r.status == "FAILED" {
			status["ErrorMessage"] = r.errMsg
		}
		out["Status"] = status
	}
}

// ---- address list imports ----

func (st *mmgrStore) createImportJob(q mmgrReq) (map[string]any, *mmgrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if r := st.byToken("importJob", mmgrStr(q.in, "ClientToken")); r != nil {
		return map[string]any{"JobId": r.id, "PreSignedUrl": r.fields["PreSignedUrl"]}, nil
	}
	list, err := st.get("addressList", mmgrStr(q.in, "AddressListId"))
	if err != nil {
		return nil, err
	}
	r, err := st.add("importJob", q.in, mmgrPick(q.in, "Name", "ImportDataFormat"), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.parent, r.status, r.uploadKey = list.id, "CREATED", strings.ReplaceAll(uuid.NewString(), "-", "")
	r.fields["PreSignedUrl"] = q.base + mmgrImportPath + r.id + "?key=" + r.uploadKey
	return map[string]any{"JobId": r.id, "PreSignedUrl": r.fields["PreSignedUrl"]}, nil
}

// runImport applies a started job's upload to its address list.
func (st *mmgrStore) runImport(r *mmgrRes) {
	defer func() {
		st.uploadBytes -= len(r.upload)
		r.upload = nil
	}()
	list := st.res[r.parent]
	if list == nil || r.upload == nil {
		r.status, r.errMsg = "FAILED", "No data was uploaded to the pre-signed URL."
		return
	}
	addrs, err := mmgrParseImport(r.upload, mmgrStr(mmgrMap(r.fields, "ImportDataFormat"), "ImportDataType"))
	if err != "" {
		r.status, r.errMsg = "FAILED", err
		return
	}
	now := time.Now().UTC()
	for _, a := range addrs {
		a = mmgrAddress(a)
		_, exists := list.members[a]
		switch {
		case exists:
			r.imported++
		case !mmgrValidAddress(a) || len(list.members) >= mmgrMaxListMembers || st.memberCount >= mmgrMaxMembers:
			r.failed++
		default:
			list.members[a] = now
			st.memberCount++
			r.imported++
		}
	}
	list.updated = now
	r.status = "COMPLETED"
}

// mmgrParseImport reads addresses from a CSV file with an "address" header
// column, or from JSON lines like {"address": "a@example.com"}.
func mmgrParseImport(data []byte, format string) ([]string, string) {
	var out []string
	if format == "JSON" {
		dec := json.NewDecoder(bytes.NewReader(data))
		for dec.More() {
			var row struct {
				Address string `json:"address"`
			}
			if err := dec.Decode(&row); err != nil {
				return nil, "The upload isn't JSON lines of {\"address\": ...} objects."
			}
			out = append(out, row.Address)
		}
		return out, ""
	}
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil || len(rows) == 0 {
		return nil, "The upload isn't a CSV file."
	}
	col := -1
	for i, h := range rows[0] {
		if strings.EqualFold(strings.TrimSpace(h), "address") {
			col = i
		}
	}
	if col < 0 {
		return nil, "The CSV header has no address column."
	}
	for _, row := range rows[1:] {
		if col < len(row) {
			out = append(out, row[col])
		}
	}
	return out, ""
}

// ---- pre-signed links ----

// handleMailManagerLinks serves archived message downloads and import
// uploads at the URLs Mail Manager hands out.
func (s *Server) handleMailManagerLinks(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if !strings.HasPrefix(path, mmgrLinkPrefix) {
		return false
	}
	st := s.mailmanager
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(path, mmgrMessagePath):
		st.mu.Lock()
		m := st.msgs[strings.TrimPrefix(path, mmgrMessagePath)]
		var raw []byte
		if m != nil {
			raw = m.raw
		}
		st.mu.Unlock()
		if m == nil {
			http.Error(w, "archived message not found", http.StatusNotFound)
			return true
		}
		w.Header().Set("Content-Type", "message/rfc822")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", "attachment")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
		return true
	case r.Method == http.MethodPut && strings.HasPrefix(path, mmgrImportPath):
		if r.ContentLength < 0 || r.ContentLength > mmgrMaxUpload {
			http.Error(w, "upload must declare a Content-Length of at most 1 MiB", http.StatusBadRequest)
			return true
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, mmgrMaxUpload+1))
		if err != nil || len(body) > mmgrMaxUpload {
			http.Error(w, "upload can't be read", http.StatusBadRequest)
			return true
		}
		status, msg := st.upload(strings.TrimPrefix(path, mmgrImportPath), r.URL.Query().Get("key"), body)
		if status != http.StatusOK {
			http.Error(w, msg, status)
			return true
		}
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}

func (st *mmgrStore) upload(jobID, key string, body []byte) (int, string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r := st.res[jobID]
	if r == nil || r.kind.id != "importJob" || subtle.ConstantTimeCompare([]byte(key), []byte(r.uploadKey)) != 1 {
		return http.StatusForbidden, "the pre-signed URL is not valid"
	}
	if r.status != "CREATED" {
		return http.StatusConflict, "the import job has already started"
	}
	if st.uploadBytes-len(r.upload)+len(body) > mmgrMaxUploadTotal {
		return http.StatusRequestEntityTooLarge, "too much pending import data"
	}
	st.uploadBytes += len(body) - len(r.upload)
	r.upload = body
	return http.StatusOK, ""
}
