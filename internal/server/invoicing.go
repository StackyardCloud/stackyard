package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Invoicing: awsJson1_0, X-Amz-Target Invoicing.*, signed invoicing,
// served through c9Serve (cloud9.go). Invoice units keep their last few
// revisions so AsOf reads see earlier rules. Invoice summaries and PDFs are
// derived from the account, billing period and invoice units
// (invoicing_billing.go); procurement portal preferences live in
// invoicing_portal.go. The generic cost-management router keeps the
// AWSBillingAndCostManagement.* traffic it serves.

const (
	invcTargetPrefix  = "Invoicing."
	invcAccount       = "123456789012"
	invcMaxUnits      = 100
	invcMaxRevisions  = 5       // revisions kept per invoice unit for AsOf reads
	invcMaxAccountIDs = 100_000 // account IDs held across every stored revision
	invcMaxTags       = 50
	invcMaxTokens     = 1000
)

var invcAPI = &c9JSONAPI{
	prefix: invcTargetPrefix, signing: "invoicing", host: "invoicing.", contentType: "application/x-amz-json-1.0",
	invalidCode: "ValidationException", model: awsmodels.InvoicingModel,
}

func init() {
	invcAPI.handlers = map[string]c9Handler{
		"BatchGetInvoiceProfile":                  (*Server).invcBatchGetInvoiceProfile,
		"CreateInvoiceUnit":                       (*Server).invcCreateUnit,
		"CreateProcurementPortalPreference":       (*Server).invcCreatePreference,
		"DeleteInvoiceUnit":                       (*Server).invcDeleteUnit,
		"DeleteProcurementPortalPreference":       (*Server).invcDeletePreference,
		"GetInvoicePDF":                           (*Server).invcGetInvoicePDF,
		"GetInvoiceUnit":                          (*Server).invcGetUnit,
		"GetProcurementPortalPreference":          (*Server).invcGetPreference,
		"ListInvoiceSummaries":                    (*Server).invcListInvoiceSummaries,
		"ListInvoiceUnits":                        (*Server).invcListUnits,
		"ListProcurementPortalPreferences":        (*Server).invcListPreferences,
		"ListProcurementPortalSuppliers":          (*Server).invcListSuppliers,
		"ListProcurementPortals":                  (*Server).invcListPortals,
		"ListTagsForResource":                     (*Server).invcListTags,
		"PutProcurementPortalPreference":          (*Server).invcPutPreference,
		"SendProcurementPortalValidation":         (*Server).invcSendValidation,
		"TagResource":                             (*Server).invcTagResource,
		"UntagResource":                           (*Server).invcUntagResource,
		"UpdateInvoiceUnit":                       (*Server).invcUpdateUnit,
		"UpdateProcurementPortalPreferenceStatus": (*Server).invcUpdatePreferenceStatus,
		"VerifyProcurementPortalValidation":       (*Server).invcVerifyValidation,
	}
}

func (s *Server) handleInvoicingJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.c9Serve(w, r, invcAPI, invcOperationByName)
}

func invcInvalid(reason, format string, args ...any) *c9Err {
	e := c9Errorf(http.StatusBadRequest, "ValidationException", format, args...)
	e.reason = reason
	return e
}

func invcNotFound(format string, args ...any) *c9Err {
	return c9Errorf(http.StatusNotFound, "ResourceNotFoundException", format, args...)
}

func invcConflict(format string, args ...any) *c9Err {
	return c9Errorf(http.StatusConflict, "ConflictException", format, args...)
}

func invcQuota(format string, args ...any) *c9Err {
	return c9Errorf(http.StatusBadRequest, "ServiceQuotaExceededException", format, args...)
}

// The model's name pattern uses look-arounds Go's regexp lacks; this is the
// same character set, with leading/trailing spaces checked separately.
var invcNameRe = regexp.MustCompile(`^[\p{L}\p{N}\p{Z}_-]+$`)

type invcRevision struct {
	description        string
	taxInheritanceOff  bool
	linked, billSource []string
	modified           time.Time
}

type invcUnit struct {
	arn, name, receiver, token string
	revisions                  []invcRevision // oldest first; never mutated in place
	tags                       map[string]string
}

func (u *invcUnit) current() invcRevision { return u.revisions[len(u.revisions)-1] }

// asOf returns the revision in force at t (zero t: the current one).
func (u *invcUnit) asOf(t time.Time) (invcRevision, bool) {
	if t.IsZero() {
		return u.current(), true
	}
	for i := len(u.revisions) - 1; i >= 0; i-- {
		if !u.revisions[i].modified.After(t) {
			return u.revisions[i], true
		}
	}
	return invcRevision{}, false
}

type invcStore struct {
	mu         sync.Mutex
	units      map[string]*invcUnit
	prefs      map[string]*invcPref
	tokens     map[string]string // client token -> ARN it created
	accountIDs int               // account IDs held across every unit revision
	pdfKey     []byte            // signs invoice PDF download URLs
}

func newInvoicingStore() *invcStore {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return &invcStore{units: map[string]*invcUnit{}, prefs: map[string]*invcPref{}, tokens: map[string]string{}, pdfKey: key}
}

func invcNewID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---- invoice units ----

// invcRule reads and checks an InvoiceUnitRule.
func invcRule(in map[string]any) ([]string, []string, *c9Err) {
	rule := c9Map(in, "Rule")
	linked, billSource := invcDedupe(c9Strings(rule, "LinkedAccounts")), invcDedupe(c9Strings(rule, "BillSourceAccounts"))
	if len(linked) == 0 && len(billSource) == 0 {
		return nil, nil, invcInvalid("invalidInput", "Rule must name at least one linked or bill source account.")
	}
	return linked, billSource, nil
}

func invcDedupe(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, v := range items {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// checkMembership refuses linked accounts already in another unit; callers hold st.mu.
func (st *invcStore) checkMembership(self string, linked []string) *c9Err {
	want := map[string]bool{}
	for _, a := range linked {
		want[a] = true
	}
	for arn, u := range st.units {
		if arn == self {
			continue
		}
		for _, a := range u.current().linked {
			if want[a] {
				return invcInvalid("accountMembershipError", "Account %s is already in invoice unit %s.", a, u.name)
			}
		}
	}
	return nil
}

// addRevision appends rev to u within the revision and account-ID budgets;
// callers hold st.mu. u.revisions is replaced, never mutated in place.
func (st *invcStore) addRevision(u *invcUnit, rev invcRevision) *c9Err {
	added := len(rev.linked) + len(rev.billSource)
	revs := append(append([]invcRevision(nil), u.revisions...), rev)
	dropped := 0
	for len(revs) > invcMaxRevisions {
		dropped += len(revs[0].linked) + len(revs[0].billSource)
		revs = revs[1:]
	}
	if st.accountIDs-dropped+added > invcMaxAccountIDs {
		return invcInvalid("maxAccountsExceeded", "Invoice units can hold at most %d account IDs in all.", invcMaxAccountIDs)
	}
	st.accountIDs += added - dropped
	u.revisions = revs
	return nil
}

func (s *Server) invcCreateUnit(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	name := c9Str(in, "Name")
	if !invcNameRe.MatchString(name) || strings.TrimSpace(name) != name {
		return nil, invcInvalid("fieldValidationFailed", "Name may hold letters, digits, spaces, - and _, and can't start or end with a space.")
	}
	linked, billSource, err := invcRule(in)
	if err != nil {
		return nil, err
	}
	tags, err := invcTags(nil, c9List(in, "ResourceTags"), nil)
	if err != nil {
		return nil, err
	}
	taxOff, _ := c9Bool(in, "TaxInheritanceDisabled")
	token := c9Str(in, "ClientToken")

	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	if arn, ok := st.tokens[token]; ok && token != "" {
		if st.units[arn] != nil {
			return map[string]any{"InvoiceUnitArn": arn}, nil
		}
	}
	for _, u := range st.units {
		if u.name == name {
			return nil, invcInvalid("duplicateInvoiceUnit", "An invoice unit named %s already exists.", name)
		}
	}
	if len(st.units) >= invcMaxUnits {
		return nil, invcInvalid("maxInvoiceUnitsExceeded", "The account already has %d invoice units.", invcMaxUnits)
	}
	if token != "" && len(st.tokens) >= invcMaxTokens {
		return nil, invcQuota("too many outstanding client tokens")
	}
	if err := st.checkMembership("", linked); err != nil {
		return nil, err
	}
	u := &invcUnit{
		arn:  "arn:aws:invoicing::" + invcAccount + ":invoice-unit/" + invcNewID(4),
		name: name, receiver: c9Str(in, "InvoiceReceiver"), token: token, tags: tags,
	}
	rev := invcRevision{description: c9Str(in, "Description"), taxInheritanceOff: taxOff, linked: linked, billSource: billSource, modified: time.Now().UTC()}
	if err := st.addRevision(u, rev); err != nil {
		return nil, err
	}
	st.units[u.arn] = u
	if token != "" {
		st.tokens[token] = u.arn
	}
	return map[string]any{"InvoiceUnitArn": u.arn}, nil
}

func (s *Server) invcUpdateUnit(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	u := st.units[c9Str(in, "InvoiceUnitArn")]
	if u == nil {
		return nil, invcNotFound("Invoice unit %s doesn't exist.", c9Str(in, "InvoiceUnitArn"))
	}
	rev := u.current()
	rev.modified = time.Now().UTC()
	if d, ok := in["Description"].(string); ok {
		rev.description = d
	}
	if v, ok := c9Bool(in, "TaxInheritanceDisabled"); ok {
		rev.taxInheritanceOff = v
	}
	if _, ok := in["Rule"]; ok {
		linked, billSource, err := invcRule(in)
		if err != nil {
			return nil, err
		}
		if err := st.checkMembership(u.arn, linked); err != nil {
			return nil, err
		}
		rev.linked, rev.billSource = linked, billSource
	}
	if err := st.addRevision(u, rev); err != nil {
		return nil, err
	}
	return map[string]any{"InvoiceUnitArn": u.arn}, nil
}

func (s *Server) invcDeleteUnit(req c9Request) (map[string]any, *c9Err) {
	arn := c9Str(req.in, "InvoiceUnitArn")
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	u := st.units[arn]
	if u == nil {
		return nil, invcNotFound("Invoice unit %s doesn't exist.", arn)
	}
	for _, p := range st.prefs {
		for _, sel := range p.unitArns() {
			if sel == arn {
				return nil, invcInvalid("invalidInput", "Invoice unit %s is selected by procurement portal preference %s.", arn, p.arn)
			}
		}
	}
	for _, r := range u.revisions {
		st.accountIDs -= len(r.linked) + len(r.billSource)
	}
	delete(st.units, arn)
	if u.token != "" {
		delete(st.tokens, u.token)
	}
	return map[string]any{"InvoiceUnitArn": arn}, nil
}

func (u *invcUnit) view(rev invcRevision) map[string]any {
	return map[string]any{
		"InvoiceUnitArn": u.arn, "InvoiceReceiver": u.receiver, "Name": u.name, "Description": rev.description,
		"TaxInheritanceDisabled": rev.taxInheritanceOff, "LastModified": c9Epoch(rev.modified),
		"Rule": map[string]any{"LinkedAccounts": c9AnyStrings(rev.linked), "BillSourceAccounts": c9AnyStrings(rev.billSource)},
	}
}

func (s *Server) invcGetUnit(req c9Request) (map[string]any, *c9Err) {
	asOf, _, err := c9Time(req.in, "AsOf")
	if err != nil {
		return nil, err
	}
	arn := c9Str(req.in, "InvoiceUnitArn")
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	u := st.units[arn]
	if u == nil {
		return nil, invcNotFound("Invoice unit %s doesn't exist.", arn)
	}
	rev, ok := u.asOf(asOf)
	if !ok {
		return nil, invcNotFound("Invoice unit %s has no revision as of %s.", arn, asOf.Format(time.RFC3339))
	}
	return u.view(rev), nil
}

func invcSet(items []string) map[string]bool {
	out := map[string]bool{}
	for _, v := range items {
		out[v] = true
	}
	return out
}

func invcAny(set map[string]bool, items []string) bool {
	for _, v := range items {
		if set[v] {
			return true
		}
	}
	return false
}

func (s *Server) invcListUnits(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	asOf, _, err := c9Time(in, "AsOf")
	if err != nil {
		return nil, err
	}
	f := c9Map(in, "Filters")
	names, receivers := invcSet(c9Strings(f, "Names")), invcSet(c9Strings(f, "InvoiceReceivers"))
	accounts, sources := invcSet(c9Strings(f, "Accounts")), invcSet(c9Strings(f, "BillSourceAccounts"))
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	arns := make([]string, 0, len(st.units))
	for arn := range st.units {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	var items []any
	for _, arn := range arns {
		u := st.units[arn]
		rev, ok := u.asOf(asOf)
		switch {
		case !ok,
			len(names) > 0 && !names[u.name],
			len(receivers) > 0 && !receivers[u.receiver],
			len(accounts) > 0 && !invcAny(accounts, rev.linked),
			len(sources) > 0 && !invcAny(sources, rev.billSource):
			continue
		}
		items = append(items, u.view(rev))
	}
	start, end, next, perr := c9Page(c9Str(in, "NextToken"), len(items), c9Int(in, "MaxResults", 100), "ValidationException")
	if perr != nil {
		perr.reason = "invalidNextToken"
		return nil, perr
	}
	return map[string]any{"InvoiceUnits": append([]any{}, items[start:end]...), "NextToken": next}, nil
}

// ---- tags ----

// invcTags merges add into cur and drops remove, returning a new map.
func invcTags(cur map[string]string, add []any, remove []string) (map[string]string, *c9Err) {
	next := make(map[string]string, len(cur)+len(add))
	for k, v := range cur {
		next[k] = v
	}
	for _, t := range add {
		tm, _ := t.(map[string]any)
		k := c9Str(tm, "Key")
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, invcInvalid("invalidInput", "Tag keys starting with aws: are reserved.")
		}
		next[k] = c9Str(tm, "Value")
	}
	for _, k := range remove {
		delete(next, k)
	}
	if len(next) > invcMaxTags {
		return nil, invcQuota("A resource can have at most %d tags.", invcMaxTags)
	}
	return next, nil
}

// tagged returns the tags of an invoice unit or preference; callers hold st.mu.
func (st *invcStore) tagged(arn string) (*map[string]string, *c9Err) {
	if u := st.units[arn]; u != nil {
		return &u.tags, nil
	}
	if p := st.prefs[arn]; p != nil {
		return &p.tags, nil
	}
	return nil, invcNotFound("Resource %s doesn't exist.", arn)
}

func (s *Server) invcTagResource(req c9Request) (map[string]any, *c9Err) {
	return s.invcRetag(c9Str(req.in, "ResourceArn"), c9List(req.in, "ResourceTags"), nil)
}

func (s *Server) invcUntagResource(req c9Request) (map[string]any, *c9Err) {
	return s.invcRetag(c9Str(req.in, "ResourceArn"), nil, c9Strings(req.in, "ResourceTagKeys"))
}

func (s *Server) invcRetag(arn string, add []any, remove []string) (map[string]any, *c9Err) {
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagged(arn)
	if err != nil {
		return nil, err
	}
	next, err := invcTags(*tags, add, remove)
	if err != nil {
		return nil, err
	}
	*tags = next
	return map[string]any{}, nil
}

func (s *Server) invcListTags(req c9Request) (map[string]any, *c9Err) {
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagged(c9Str(req.in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ResourceTags": c9TagList(*tags, "Key", "Value")}, nil
}
