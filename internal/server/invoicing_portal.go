package server

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"time"
)

// Procurement portal preferences: e-invoice delivery and purchase-order
// retrieval through SAP Business Network or Coupa. Enabled features start
// PENDING_VERIFICATION; SendProcurementPortalValidation "emails" the contact
// a six-digit code and VerifyProcurementPortalValidation with that code moves
// them to VALIDATED. The code is the first four bytes of
// SHA-256("<preference ARN>#<send count>") as a big-endian integer, modulo
// 1,000,000, zero-padded. Shared secrets are stored but never returned.

const (
	invcMaxPrefs      = 20
	invcCodeLifetime  = 24 * time.Hour
	invcPortalSAP     = "sap-business-network"
	invcPortalCoupa   = "coupa"
	invcPrefArnPrefix = "arn:aws:invoicing::" + invcAccount + ":procurement-portal-preference/"
)

type invcPortal struct {
	id, name, display string
	docTypes, attach  []string
	suppliers         []invcSupplier
}

type invcSupplier struct{ id, seller, country, env string }

func invcSuppliers(prefix string) []invcSupplier {
	sellers := []struct{ name, country string }{
		{"AWS_INC", "US"}, {"AWS_EMEA_SARL", "LU"}, {"AWS_JAPAN_GK", "JP"},
	}
	var out []invcSupplier
	for i, s := range sellers {
		id := fmt.Sprintf("%s%010d", prefix, 1000+i)
		out = append(out, invcSupplier{id, s.name, s.country, "PROD"}, invcSupplier{id + "-T", s.name, s.country, "TEST"})
	}
	return out
}

var invcPortals = []invcPortal{
	{
		id: invcPortalCoupa, name: "COUPA", display: "Coupa",
		docTypes:  []string{"AWS_CLOUD_INVOICE", "AWS_CLOUD_CREDIT_MEMO", "AWS_REQUEST_FOR_PAYMENT"},
		attach:    []string{"INVOICE_PDF", "RFP_PDF"},
		suppliers: invcSuppliers("CP"),
	},
	{
		id: invcPortalSAP, name: "SAP_BUSINESS_NETWORK", display: "SAP Business Network",
		docTypes:  []string{"AWS_CLOUD_INVOICE", "AWS_CLOUD_CREDIT_MEMO", "AWS_MARKETPLACE_INVOICE", "AWS_MARKETPLACE_CREDIT_MEMO"},
		attach:    []string{"INVOICE_PDF"},
		suppliers: invcSuppliers("AN"),
	},
}

func invcPortalByName(name string) *invcPortal {
	for i := range invcPortals {
		if invcPortals[i].name == name || invcPortals[i].id == name {
			return &invcPortals[i]
		}
	}
	return nil
}

func (p *invcPortal) hasSupplier(id, env string) bool {
	for _, s := range p.suppliers {
		if s.id == id && s.env == env {
			return true
		}
	}
	return false
}

// invcPrefConfig are the members PutProcurementPortalPreference replaces.
var invcPrefConfig = []string{
	"Selector", "ProcurementPortalSharedSecret", "ProcurementPortalInstanceEndpoint", "TestEnvPreference",
	"EinvoiceDeliveryEnabled", "EinvoiceDeliveryPreference", "PurchaseOrderRetrievalEnabled",
	"MarketplacePunchOutEnabled", "MarketplacePunchOutPreference", "Contacts",
}

type invcPref struct {
	arn, id, portal, buyerID, supplierID, token string
	config                                      map[string]any // model-conformed; replaced, never mutated
	einvStatus, einvReason, poStatus, poReason  string
	verified                                    bool
	version                                     int
	created, updated                            time.Time
	tags                                        map[string]string
	sends                                       int
	code                                        string
	codeExpiry                                  time.Time
}

func (p *invcPref) unitArns() []string {
	return c9Strings(c9Map(p.config, "Selector"), "InvoiceUnitArns")
}

func (p *invcPref) enabled(key string) bool {
	v, _ := c9Bool(p.config, key)
	return v
}

func invcCode(arn string, sends int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s#%d", arn, sends)))
	return fmt.Sprintf("%06d", binary.BigEndian.Uint32(sum[:4])%1_000_000)
}

// prefConfig picks and checks the replaceable members of a create or put;
// callers hold st.mu. self is the preference being replaced, if any.
func (st *invcStore) prefConfig(in map[string]any, portal *invcPortal, self string) (map[string]any, *c9Err) {
	cfg := map[string]any{}
	for _, k := range invcPrefConfig {
		if v, ok := in[k]; ok {
			cfg[k] = v
		}
	}
	einv, _ := c9Bool(cfg, "EinvoiceDeliveryEnabled")
	pref := c9Map(cfg, "EinvoiceDeliveryPreference")
	if einv && pref == nil {
		return nil, invcInvalid("invalidInput", "EinvoiceDeliveryPreference is required when e-invoice delivery is enabled.")
	}
	test := c9Map(cfg, "TestEnvPreference")
	if c9Str(pref, "ConnectionTestingMethod") == "TEST_ENV_REPLAY_TEST" && test == nil {
		return nil, invcInvalid("invalidInput", "TestEnvPreference is required for TEST_ENV_REPLAY_TEST.")
	}
	if test != nil && !portal.hasSupplier(c9Str(test, "SupplierIdentifier"), "TEST") {
		return nil, invcInvalid("invalidInput", "%s isn't a test supplier on %s.", c9Str(test, "SupplierIdentifier"), portal.display)
	}
	if punch, _ := c9Bool(cfg, "MarketplacePunchOutEnabled"); punch && c9Map(cfg, "MarketplacePunchOutPreference") == nil {
		return nil, invcInvalid("invalidInput", "MarketplacePunchOutPreference is required when punch-out is enabled.")
	}
	for _, arn := range c9Strings(c9Map(cfg, "Selector"), "InvoiceUnitArns") {
		if st.units[arn] == nil {
			return nil, invcInvalid("invalidInput", "Invoice unit %s doesn't exist.", arn)
		}
		for _, other := range st.prefs {
			if other.arn == self {
				continue
			}
			for _, a := range other.unitArns() {
				if a == arn {
					return nil, invcConflict("Invoice unit %s is already selected by %s.", arn, other.arn)
				}
			}
		}
	}
	return cfg, nil
}

// resetStatuses puts every enabled feature back to PENDING_VERIFICATION.
func (p *invcPref) resetStatuses() {
	p.einvStatus, p.einvReason, p.poStatus, p.poReason, p.verified, p.code = "", "", "", "", false, ""
	if p.enabled("EinvoiceDeliveryEnabled") {
		p.einvStatus, p.einvReason = "PENDING_VERIFICATION", "Waiting for the contact to verify the connection."
	}
	if p.enabled("PurchaseOrderRetrievalEnabled") {
		p.poStatus, p.poReason = "PENDING_VERIFICATION", "Waiting for the contact to verify the connection."
	}
}

func (s *Server) invcCreatePreference(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	portal := invcPortalByName(c9Str(in, "ProcurementPortalName"))
	if portal == nil {
		return nil, invcInvalid("invalidInput", "unknown procurement portal")
	}
	supplier := c9Str(in, "SupplierIdentifier")
	if !portal.hasSupplier(supplier, "PROD") {
		return nil, invcInvalid("invalidInput", "%s isn't a supplier on %s.", supplier, portal.display)
	}
	tags, err := invcTags(nil, c9List(in, "ResourceTags"), nil)
	if err != nil {
		return nil, err
	}
	token := c9Str(in, "ClientToken")
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	if arn, ok := st.tokens[token]; ok && token != "" && st.prefs[arn] != nil {
		return map[string]any{"ProcurementPortalPreferenceArn": arn}, nil
	}
	buyer := c9Str(in, "BuyerIdentifier")
	for _, p := range st.prefs {
		if p.portal == portal.name && p.buyerID == buyer && p.supplierID == supplier {
			return nil, invcConflict("A preference for buyer %s and supplier %s on %s already exists.", buyer, supplier, portal.display)
		}
	}
	if len(st.prefs) >= invcMaxPrefs {
		return nil, invcQuota("The account already has %d procurement portal preferences.", invcMaxPrefs)
	}
	if token != "" && len(st.tokens) >= invcMaxTokens {
		return nil, invcQuota("too many outstanding client tokens")
	}
	cfg, err := st.prefConfig(in, portal, "")
	if err != nil {
		return nil, err
	}
	id := invcNewID(16)
	now := time.Now().UTC()
	p := &invcPref{
		arn: invcPrefArnPrefix + id, id: id, portal: portal.name, buyerID: buyer, supplierID: supplier, token: token,
		config: cfg, version: 1, created: now, updated: now, tags: tags,
	}
	p.resetStatuses()
	st.prefs[p.arn] = p
	if token != "" {
		st.tokens[token] = p.arn
	}
	return map[string]any{"ProcurementPortalPreferenceArn": p.arn}, nil
}

// pref finds a preference; callers hold st.mu.
func (st *invcStore) pref(arn string) (*invcPref, *c9Err) {
	if p := st.prefs[arn]; p != nil {
		return p, nil
	}
	return nil, invcNotFound("Procurement portal preference %s doesn't exist.", arn)
}

func (p *invcPref) touch() {
	p.version++
	p.updated = time.Now().UTC()
}

func (s *Server) invcPutPreference(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pref(c9Str(in, "ProcurementPortalPreferenceArn"))
	if err != nil {
		return nil, err
	}
	cfg, err := st.prefConfig(in, invcPortalByName(p.portal), p.arn)
	if err != nil {
		return nil, err
	}
	p.config = cfg
	p.resetStatuses()
	p.touch()
	return map[string]any{"ProcurementPortalPreferenceArn": p.arn}, nil
}

func (s *Server) invcDeletePreference(req c9Request) (map[string]any, *c9Err) {
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pref(c9Str(req.in, "ProcurementPortalPreferenceArn"))
	if err != nil {
		return nil, err
	}
	delete(st.prefs, p.arn)
	if p.token != "" {
		delete(st.tokens, p.token)
	}
	return map[string]any{"ProcurementPortalPreferenceArn": p.arn}, nil
}

// view renders a preference; full adds the Get-only members. Shared
// secrets are never returned.
func (p *invcPref) view(full bool) map[string]any {
	out := map[string]any{
		"AwsAccountId": invcAccount, "ProcurementPortalPreferenceArn": p.arn, "ProcurementPortalName": p.portal,
		"BuyerDomain": "NetworkID", "BuyerIdentifier": p.buyerID, "SupplierDomain": "NetworkID", "SupplierIdentifier": p.supplierID,
		"EinvoiceDeliveryEnabled": p.enabled("EinvoiceDeliveryEnabled"), "PurchaseOrderRetrievalEnabled": p.enabled("PurchaseOrderRetrievalEnabled"),
		"MarketplacePunchOutEnabled": p.enabled("MarketplacePunchOutEnabled"),
		"Version":                    float64(p.version), "CreateDate": c9Epoch(p.created), "LastUpdateDate": c9Epoch(p.updated),
	}
	if sel := c9Map(p.config, "Selector"); sel != nil {
		out["Selector"] = cncCopy(sel)
	}
	for k, v := range map[string]string{
		"EinvoiceDeliveryPreferenceStatus": p.einvStatus, "EinvoiceDeliveryPreferenceStatusReason": p.einvReason,
		"PurchaseOrderRetrievalPreferenceStatus": p.poStatus, "PurchaseOrderRetrievalPreferenceStatusReason": p.poReason,
	} {
		if v != "" {
			out[k] = v
		}
	}
	if !full {
		return out
	}
	for _, k := range []string{"ProcurementPortalInstanceEndpoint", "EinvoiceDeliveryPreference", "MarketplacePunchOutPreference", "Contacts"} {
		if v, ok := p.config[k]; ok {
			out[k] = cncCopy(v)
		}
	}
	if test := c9Map(p.config, "TestEnvPreference"); test != nil {
		t := cncCopyMap(test)
		delete(t, "ProcurementPortalSharedSecret")
		out["TestEnvPreference"] = t
	}
	if p.enabled("PurchaseOrderRetrievalEnabled") {
		out["PurchaseOrderRetrievalEndpoint"] = "https://invoicing.us-east-1.amazonaws.com/procurement-portal/" + p.id + "/purchase-orders"
	}
	return out
}

func (s *Server) invcGetPreference(req c9Request) (map[string]any, *c9Err) {
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pref(c9Str(req.in, "ProcurementPortalPreferenceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ProcurementPortalPreference": p.view(true)}, nil
}

func (s *Server) invcListPreferences(req c9Request) (map[string]any, *c9Err) {
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	arns := make([]string, 0, len(st.prefs))
	for arn := range st.prefs {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	start, end, next, err := c9Page(c9Str(req.in, "NextToken"), len(arns), c9Int(req.in, "MaxResults", 100), "ValidationException")
	if err != nil {
		err.reason = "invalidNextToken"
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, arn := range arns[start:end] {
		out = append(out, st.prefs[arn].view(false))
	}
	return map[string]any{"ProcurementPortalPreferences": out, "NextToken": next}, nil
}

func (s *Server) invcSendValidation(req c9Request) (map[string]any, *c9Err) {
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pref(c9Str(req.in, "ProcurementPortalPreferenceArn"))
	if err != nil {
		return nil, err
	}
	if !p.enabled("EinvoiceDeliveryEnabled") && !p.enabled("PurchaseOrderRetrievalEnabled") {
		return nil, invcInvalid("invalidInput", "Neither e-invoice delivery nor purchase-order retrieval is enabled.")
	}
	email := ""
	if contacts := c9List(p.config, "Contacts"); len(contacts) > 0 {
		c, _ := contacts[0].(map[string]any)
		email = c9Str(c, "Email")
	}
	p.sends++
	p.code, p.codeExpiry = invcCode(p.arn, p.sends), time.Now().Add(invcCodeLifetime)
	reason := "A validation code was sent to " + email + "."
	if p.einvStatus != "" {
		p.einvStatus, p.einvReason = "PENDING_VERIFICATION", reason
	}
	if p.poStatus != "" {
		p.poStatus, p.poReason = "PENDING_VERIFICATION", reason
	}
	p.touch()
	return map[string]any{"ProcurementPortalPreferenceArn": p.arn}, nil
}

func (s *Server) invcVerifyValidation(req c9Request) (map[string]any, *c9Err) {
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pref(c9Str(req.in, "ProcurementPortalPreferenceArn"))
	if err != nil {
		return nil, err
	}
	if p.code == "" || time.Now().After(p.codeExpiry) {
		return nil, invcInvalid("invalidInput", "No validation code is outstanding; call SendProcurementPortalValidation first.")
	}
	if c9Str(req.in, "Code") != p.code {
		return nil, invcInvalid("invalidInput", "The validation code is not correct.")
	}
	p.code, p.verified = "", true
	if p.einvStatus != "" {
		p.einvStatus, p.einvReason = "VALIDATED", "The connection was verified."
	}
	if p.poStatus != "" {
		p.poStatus, p.poReason = "VALIDATED", "The connection was verified."
	}
	p.touch()
	return map[string]any{"ProcurementPortalPreferenceArn": p.arn}, nil
}

// invcStatusAllowed reports whether a feature may move to status; VALIDATED,
// ACTIVE and the TEST_* statuses need a verified connection.
func invcStatusAllowed(status string, verified bool) bool {
	return status == "PENDING_VERIFICATION" || status == "SUSPENDED" || verified
}

func (s *Server) invcUpdatePreferenceStatus(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pref(c9Str(in, "ProcurementPortalPreferenceArn"))
	if err != nil {
		return nil, err
	}
	einv, po := c9Str(in, "EinvoiceDeliveryPreferenceStatus"), c9Str(in, "PurchaseOrderRetrievalPreferenceStatus")
	if einv == "" && po == "" {
		return nil, invcInvalid("invalidInput", "Set EinvoiceDeliveryPreferenceStatus or PurchaseOrderRetrievalPreferenceStatus.")
	}
	for _, c := range []struct {
		status, feature string
		on              bool
	}{{einv, "e-invoice delivery", p.enabled("EinvoiceDeliveryEnabled")}, {po, "purchase-order retrieval", p.enabled("PurchaseOrderRetrievalEnabled")}} {
		if c.status == "" {
			continue
		}
		if !c.on {
			return nil, invcInvalid("invalidInput", "%s isn't enabled for this preference.", c.feature)
		}
		if !invcStatusAllowed(c.status, p.verified) {
			return nil, invcConflict("%s can't move to %s before the connection is verified.", c.feature, c.status)
		}
	}
	if einv != "" {
		p.einvStatus, p.einvReason = einv, c9Str(in, "EinvoiceDeliveryPreferenceStatusReason")
	}
	if po != "" {
		p.poStatus, p.poReason = po, c9Str(in, "PurchaseOrderRetrievalPreferenceStatusReason")
	}
	p.touch()
	return map[string]any{"ProcurementPortalPreferenceArn": p.arn}, nil
}

// ---- portals ----

func (s *Server) invcListPortals(req c9Request) (map[string]any, *c9Err) {
	start, end, next, err := c9Page(c9Str(req.in, "NextToken"), len(invcPortals), c9Int(req.in, "MaxResults", 100), "ValidationException")
	if err != nil {
		err.reason = "invalidNextToken"
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, p := range invcPortals[start:end] {
		out = append(out, map[string]any{
			"PortalIdentifier": p.id, "PortalName": p.name, "PortalDisplayName": p.display,
			"DefaultFeatureConfigurations": map[string]any{"InvoiceConfiguration": map[string]any{
				"DocumentTypes": c9AnyStrings(p.docTypes), "AttachmentTypes": c9AnyStrings(p.attach),
			}},
		})
	}
	return map[string]any{"ProcurementPortals": out, "NextToken": next}, nil
}

func (s *Server) invcListSuppliers(req c9Request) (map[string]any, *c9Err) {
	id := c9Str(req.in, "PortalIdentifier")
	var portal *invcPortal
	for i := range invcPortals {
		if invcPortals[i].id == id {
			portal = &invcPortals[i]
		}
	}
	if portal == nil {
		return nil, invcNotFound("Procurement portal %s doesn't exist.", id)
	}
	start, end, next, err := c9Page(c9Str(req.in, "NextToken"), len(portal.suppliers), c9Int(req.in, "MaxResults", 100), "ValidationException")
	if err != nil {
		err.reason = "invalidNextToken"
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, sp := range portal.suppliers[start:end] {
		out = append(out, map[string]any{"SupplierIdentifier": sp.id, "SellerOfRecord": sp.seller, "CountryCode": sp.country, "Environment": sp.env})
	}
	return map[string]any{"ProcurementPortalSuppliers": out, "NextToken": next}, nil
}
