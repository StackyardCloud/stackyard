package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Tax Settings: REST JSON, signed tax, every operation POST /<Op> with a
// body-only input, served through acctServeREST/budRun. Registrations and
// exemptions belong to the caller (the Organizations management account) or
// its member accounts, resolved through the Organizations store before the
// Tax Settings lock is taken. Verification documents are stored (inline or
// copied from Stackyard S3) and GetTaxRegistrationDocument writes them back
// to S3.

const (
	// ponytail: a 2 MiB body caps inline documents and exemption certificates
	// well under the model's 5 MiB / 4 MiB; raise it if real files need it.
	taxsMaxBody      = 2 << 20
	taxsMaxDoc       = 1 << 20  // one stored verification document
	taxsMaxDocBytes  = 16 << 20 // all stored documents
	taxsMaxSupp      = 100      // supplemental registrations
	taxsMaxExemption = 50       // exemptions per account
)

var taxsAPI = &budJSONAPI{
	signing: "tax", contentType: "application/json",
	invalidCode: "ValidationException", model: awsmodels.TaxSettingsModel, maxBody: taxsMaxBody,
	respondError: taxsRespondError, s3Caller: true,
}

const taxsServicePrincipal = "tax.amazonaws.com"

// taxsRespondError writes a REST JSON error in the Tax Settings model's
// shape: lower-case message plus the errorCode its exceptions require.
func taxsRespondError(w http.ResponseWriter, e *budErr) {
	errorCode := map[string]string{
		"ValidationException": "FieldValidationFailed", "ConflictException": "Conflict",
		"ResourceNotFoundException": "ResourceNotFound",
	}[e.code]
	body := map[string]any{"message": e.msg}
	if errorCode != "" {
		body["errorCode"] = errorCode
	}
	for k, v := range e.extra {
		body[k] = v
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

var taxsPaths = func() map[string]string {
	out := make(map[string]string, len(taxsOperations))
	for _, op := range taxsOperations {
		out["/"+op.Name] = op.Name
	}
	return out
}()

func init() {
	taxsAPI.handlers = map[string]budHandler{
		"PutTaxRegistration":                (*Server).taxsPutTaxRegistration,
		"BatchPutTaxRegistration":           (*Server).taxsBatchPutTaxRegistration,
		"GetTaxRegistration":                (*Server).taxsGetTaxRegistration,
		"ListTaxRegistrations":              (*Server).taxsListTaxRegistrations,
		"DeleteTaxRegistration":             (*Server).taxsDeleteTaxRegistration,
		"BatchDeleteTaxRegistration":        (*Server).taxsBatchDeleteTaxRegistration,
		"GetTaxRegistrationDocument":        (*Server).taxsGetTaxRegistrationDocument,
		"PutSupplementalTaxRegistration":    (*Server).taxsPutSupplemental,
		"ListSupplementalTaxRegistrations":  (*Server).taxsListSupplemental,
		"DeleteSupplementalTaxRegistration": (*Server).taxsDeleteSupplemental,
		"GetTaxExemptionTypes":              (*Server).taxsGetTaxExemptionTypes,
		"PutTaxExemption":                   (*Server).taxsPutTaxExemption,
		"BatchGetTaxExemptions":             (*Server).taxsBatchGetTaxExemptions,
		"ListTaxExemptions":                 (*Server).taxsListTaxExemptions,
		"GetTaxInheritance":                 (*Server).taxsGetTaxInheritance,
		"PutTaxInheritance":                 (*Server).taxsPutTaxInheritance,
	}
}

func (s *Server) handleTaxSettingsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.acctServeREST(w, r, taxsAPI, taxsPaths, "tax.")
}

func taxsInvalid(format string, args ...any) *budErr {
	return budErrorf("ValidationException", format, args...)
}

// taxsAccount resolves accountId (empty: the caller) against the
// organization; it takes and releases the Organizations lock.
func (s *Server) taxsAccount(id string) (string, *budErr) {
	if id == "" {
		return acctCaller, nil
	}
	if _, ok := s.organizations.account(id); !ok {
		return "", taxsInvalid("Account %s is not your account or a member of your organization.", id)
	}
	return id, nil
}

// ---- registrations ----

// taxsLoadDocs turns verificationDetails.taxRegistrationDocuments into
// stored documents, reading S3 sources now (S3's lock, never ours).
func (s *Server) taxsLoadDocs(c s3Caller, entry map[string]any) ([]taxsDoc, *budErr) {
	var docs []taxsDoc
	for _, item := range budList(budMap(entry, "verificationDetails"), "taxRegistrationDocuments") {
		d, _ := item.(map[string]any)
		var name string
		var body []byte
		if f := budMap(d, "file"); f != nil {
			raw, err := base64.StdEncoding.DecodeString(budStr(f, "fileContent"))
			if err != nil || len(raw) == 0 {
				return nil, taxsInvalid("fileContent must be non-empty base64")
			}
			name, body = budStr(f, "fileName"), raw
		} else if loc := budMap(d, "s3Location"); loc != nil {
			if !s.authorizeServiceS3As(c, taxsServicePrincipal, budStr(loc, "bucket"), budStr(loc, "key"), "s3:GetObject") {
				return nil, budErrorf("AccessDeniedException", "Access denied reading s3://%s/%s", budStr(loc, "bucket"), budStr(loc, "key"))
			}
			obj, err := s.s3.GetObject(budStr(loc, "bucket"), budStr(loc, "key"))
			if err != nil {
				return nil, taxsInvalid("s3://%s/%s can't be read", budStr(loc, "bucket"), budStr(loc, "key"))
			}
			name, body = path.Base(obj.Key), obj.Body
		} else {
			return nil, taxsInvalid("each tax registration document needs a file or an s3Location")
		}
		if len(body) > taxsMaxDoc {
			return nil, taxsInvalid("tax registration documents can be at most %d bytes", taxsMaxDoc)
		}
		docs = append(docs, taxsDoc{token: budNewID(), name: name, body: body})
	}
	return docs, nil
}

// taxsEntry checks a registration entry and returns the copy that is stored
// (verification documents travel separately).
func taxsEntry(in map[string]any) (map[string]any, *budErr) {
	entry := budCloneMap(budMap(in, "taxRegistrationEntry"))
	if budStr(entry, "legalName") == "" || budMap(entry, "legalAddress") == nil {
		return nil, taxsInvalid("taxRegistrationEntry needs a legalName and a legalAddress")
	}
	delete(entry, "verificationDetails")
	return entry, nil
}

func (s *Server) taxsPutTaxRegistration(in map[string]any) (map[string]any, *budErr) {
	account, err := s.taxsAccount(budStr(in, "accountId"))
	if err != nil {
		return nil, err
	}
	entry, err := taxsEntry(in)
	if err != nil {
		return nil, err
	}
	docs, err := s.taxsLoadDocs(s3CallerIn(in), budMap(in, "taxRegistrationEntry"))
	if err != nil {
		return nil, err
	}
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, err := st.put(account, entry, docs)
	if err != nil {
		return nil, err
	}
	return map[string]any{"status": reg.status}, nil
}

func (s *Server) taxsBatchPutTaxRegistration(in map[string]any) (map[string]any, *budErr) {
	entry, err := taxsEntry(in)
	if err != nil {
		return nil, err
	}
	docs, err := s.taxsLoadDocs(s3CallerIn(in), budMap(in, "taxRegistrationEntry"))
	if err != nil {
		return nil, err
	}
	// Duplicates are dropped and each account's copy is built before the
	// lock, so the locked loop does at most one put per roster account.
	ids := taxsUnique(budStrings(in, "accountIds")) // at most 550 (model)
	errs, status := []any{}, ""
	type taxsPending struct {
		id    string
		entry map[string]any
		docs  []taxsDoc
	}
	var pending []taxsPending
	for _, id := range ids {
		if _, err := s.taxsAccount(id); err != nil {
			errs = append(errs, map[string]any{"accountId": id, "code": err.code, "message": err.msg})
			continue
		}
		pending = append(pending, taxsPending{id, budCloneMap(entry), taxsCopyDocs(docs)})
	}
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, p := range pending {
		id := p.id
		reg, perr := st.put(id, p.entry, p.docs)
		if perr != nil {
			errs = append(errs, map[string]any{"accountId": id, "code": perr.code, "message": perr.msg})
			continue
		}
		status = reg.status
	}
	out := map[string]any{"errors": errs}
	if status != "" {
		out["status"] = status
	}
	return out, nil
}

// taxsUnique drops repeated IDs, keeping the first.
func taxsUnique(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := ids[:0:0]
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// taxsCopyDocs gives each account its own documents and access tokens.
func taxsCopyDocs(docs []taxsDoc) []taxsDoc {
	out := make([]taxsDoc, len(docs))
	for i, d := range docs {
		out[i] = taxsDoc{token: budNewID(), name: d.name, body: d.body}
	}
	return out
}

func (s *Server) taxsGetTaxRegistration(in map[string]any) (map[string]any, *budErr) {
	account, err := s.taxsAccount(budStr(in, "accountId"))
	if err != nil {
		return nil, err
	}
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	reg := st.regs[account]
	if reg == nil {
		return nil, budErrorf("ResourceNotFoundException", "Account %s has no tax registration.", account)
	}
	return map[string]any{"taxRegistration": reg.out(false)}, nil
}

func (s *Server) taxsListTaxRegistrations(in map[string]any) (map[string]any, *budErr) {
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	accounts := st.registeredAccounts()
	start, end, next, err := budPage(budStr(in, "nextToken"), len(accounts), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	details := []any{}
	for _, id := range accounts[start:end] {
		details = append(details, map[string]any{"accountId": id, "taxRegistration": st.regs[id].out(true)})
	}
	return map[string]any{"accountDetails": details, "nextToken": next}, nil
}

func (s *Server) taxsDeleteTaxRegistration(in map[string]any) (map[string]any, *budErr) {
	account, err := s.taxsAccount(budStr(in, "accountId"))
	if err != nil {
		return nil, err
	}
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.remove(account) {
		return nil, budErrorf("ResourceNotFoundException", "Account %s has no tax registration.", account)
	}
	return map[string]any{}, nil
}

func (s *Server) taxsBatchDeleteTaxRegistration(in map[string]any) (map[string]any, *budErr) {
	ids := taxsUnique(budStrings(in, "accountIds")) // at most 5 (model)
	known := map[string]bool{}
	for _, id := range ids {
		_, err := s.taxsAccount(id)
		known[id] = err == nil
	}
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	errs := []any{}
	for _, id := range ids {
		if !known[id] || !st.remove(id) {
			errs = append(errs, map[string]any{"accountId": id, "code": "ResourceNotFoundException", "message": "Account " + id + " has no tax registration."})
		}
	}
	return map[string]any{"errors": errs}, nil
}

// taxsGetTaxRegistrationDocument copies a stored document to an S3 bucket.
// Stackyard has no AWS-hosted download location to presign, so a
// destinationS3Location is required.
func (s *Server) taxsGetTaxRegistrationDocument(in map[string]any) (map[string]any, *budErr) {
	meta, dest := budMap(in, "taxDocumentMetadata"), budMap(in, "destinationS3Location")
	if dest == nil {
		return nil, taxsInvalid("Stackyard returns tax registration documents only to a destinationS3Location.")
	}
	st := s.taxSettings
	st.mu.Lock()
	doc, ok := st.document(budStr(meta, "taxDocumentAccessToken"))
	st.mu.Unlock()
	if !ok || doc.name != budStr(meta, "taxDocumentName") {
		return nil, taxsInvalid("No tax registration document matches that access token and name.")
	}
	bucket := budStr(dest, "bucket")
	key := doc.name
	if prefix := strings.Trim(budStr(dest, "prefix"), "/"); prefix != "" {
		key = prefix + "/" + doc.name
	}
	if !s.authorizeServiceS3As(s3CallerIn(in), taxsServicePrincipal, bucket, key, "s3:PutObject") {
		return nil, budErrorf("AccessDeniedException", "Access denied writing s3://%s/%s", bucket, key)
	}
	// doc.body is never mutated once stored, so writing it unlocked is safe.
	if _, err := s.s3.PutObject(bucket, key, "application/octet-stream", doc.body, nil, "", ""); err != nil {
		return nil, taxsInvalid("The document can't be written to s3://%s: %v", bucket, err)
	}
	return map[string]any{"destinationFilePath": fmt.Sprintf("s3://%s/%s", bucket, key)}, nil
}

// ---- supplemental registrations (caller only) ----

func (s *Server) taxsPutSupplemental(in map[string]any) (map[string]any, *budErr) {
	entry := budCloneMap(budMap(in, "taxRegistrationEntry"))
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	country := budStr(budMap(entry, "address"), "countryCode")
	for _, sp := range st.supplemental {
		if budStr(sp, "registrationId") == budStr(entry, "registrationId") && budStr(budMap(sp, "address"), "countryCode") == country {
			return nil, budErrorf("ConflictException", "Supplemental registration %s already exists for %s.", budStr(entry, "registrationId"), country)
		}
	}
	if len(st.supplemental) >= taxsMaxSupp {
		return nil, taxsInvalid("An account can have at most %d supplemental tax registrations.", taxsMaxSupp)
	}
	entry["authorityId"], entry["status"] = budNewID(), "Verified"
	st.supplemental = append(st.supplemental, entry)
	return map[string]any{"authorityId": entry["authorityId"], "status": entry["status"]}, nil
}

func (s *Server) taxsListSupplemental(in map[string]any) (map[string]any, *budErr) {
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	start, end, next, err := budPage(budStr(in, "nextToken"), len(st.supplemental), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	list := []any{}
	for _, sp := range st.supplemental[start:end] {
		list = append(list, budCloneMap(sp))
	}
	return map[string]any{"taxRegistrations": list, "nextToken": next}, nil
}

func (s *Server) taxsDeleteSupplemental(in map[string]any) (map[string]any, *budErr) {
	id := budStr(in, "authorityId")
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, sp := range st.supplemental {
		if sp["authorityId"] == id {
			st.supplemental = append(st.supplemental[:i:i], st.supplemental[i+1:]...)
			return map[string]any{}, nil
		}
	}
	return nil, budErrorf("ResourceNotFoundException", "Supplemental tax registration %s was not found.", id)
}

// ---- exemptions and inheritance ----

func (s *Server) taxsGetTaxExemptionTypes(map[string]any) (map[string]any, *budErr) {
	list := make([]any, 0, len(taxsExemptionTypes))
	for _, t := range taxsExemptionTypes {
		list = append(list, t.out())
	}
	return map[string]any{"taxExemptionTypes": list}, nil
}

func (s *Server) taxsPutTaxExemption(in map[string]any) (map[string]any, *budErr) {
	authority := budMap(in, "authority")
	country, state := strings.ToUpper(budStr(authority, "country")), budStr(authority, "state")
	typ := taxsExemptionType(budStr(in, "exemptionType"))
	if typ == nil {
		return nil, taxsInvalid("%s is not a tax exemption type; see GetTaxExemptionTypes.", budStr(in, "exemptionType"))
	}
	if !typ.applies(country, state) {
		return nil, taxsInvalid("%s exemptions don't apply in %s %s.", typ.name, country, state)
	}
	cert := budMap(in, "exemptionCertificate")
	if raw, err := base64.StdEncoding.DecodeString(budStr(cert, "documentFile")); err != nil || len(raw) == 0 {
		return nil, taxsInvalid("documentFile must be non-empty base64")
	}
	ids := taxsUnique(budStrings(in, "accountIds")) // at most 550 (model)
	for _, id := range ids {
		if _, err := s.taxsAccount(id); err != nil {
			return nil, err
		}
	}
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	// Check every account's cap first so the write is all or nothing.
	for _, id := range ids {
		if !st.exemptionFits(id, country, state) {
			return nil, taxsInvalid("Account %s already has %d tax exemptions.", id, taxsMaxExemption)
		}
	}
	caseID := "case-" + budNewID()
	for _, id := range ids {
		st.putExemption(id, country, state, typ, caseID)
	}
	return map[string]any{"caseId": caseID}, nil
}

func (s *Server) taxsBatchGetTaxExemptions(in map[string]any) (map[string]any, *budErr) {
	ids := taxsUnique(budStrings(in, "accountIds")) // at most 5 (model)
	known := map[string]bool{}
	for _, id := range ids {
		_, err := s.taxsAccount(id)
		known[id] = err == nil
	}
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	details, failed := map[string]any{}, []any{}
	for _, id := range ids {
		if !known[id] {
			failed = append(failed, id)
			continue
		}
		details[id] = st.exemptionDetails(id)
	}
	return map[string]any{"taxExemptionDetailsMap": details, "failedAccounts": failed}, nil
}

func (s *Server) taxsListTaxExemptions(in map[string]any) (map[string]any, *budErr) {
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	accounts := st.exemptAccounts()
	start, end, next, err := budPage(budStr(in, "nextToken"), len(accounts), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	details := map[string]any{}
	for _, id := range accounts[start:end] {
		details[id] = st.exemptionDetails(id)
	}
	return map[string]any{"taxExemptionDetailsMap": details, "nextToken": next}, nil
}

func (s *Server) taxsGetTaxInheritance(map[string]any) (map[string]any, *budErr) {
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	return map[string]any{"heritageStatus": st.heritage}, nil
}

// taxsPutTaxInheritance opts member accounts in or out of inheriting the
// management account's tax settings; opting in needs a registration to inherit.
func (s *Server) taxsPutTaxInheritance(in map[string]any) (map[string]any, *budErr) {
	status := budStr(in, "heritageStatus")
	if status == "" {
		return nil, taxsInvalid("heritageStatus is required")
	}
	st := s.taxSettings
	st.mu.Lock()
	defer st.mu.Unlock()
	if status == "OptIn" && st.regs[acctCaller] == nil {
		return nil, budErrorf("ResourceNotFoundException", "The management account has no tax registration for member accounts to inherit.")
	}
	st.heritage = status
	return map[string]any{}, nil
}
