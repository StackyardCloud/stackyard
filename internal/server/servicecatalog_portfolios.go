package server

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

func init() {
	svcatHandlers = map[string]svcatHandler{}
	for name, h := range map[string]svcatHandler{
		"CreatePortfolio":                    (*serviceCatalogStore).createPortfolio,
		"DescribePortfolio":                  (*serviceCatalogStore).describePortfolio,
		"UpdatePortfolio":                    (*serviceCatalogStore).updatePortfolio,
		"DeletePortfolio":                    (*serviceCatalogStore).deletePortfolio,
		"ListPortfolios":                     (*serviceCatalogStore).listPortfolios,
		"ListPortfoliosForProduct":           (*serviceCatalogStore).listPortfoliosForProduct,
		"CreatePortfolioShare":               (*serviceCatalogStore).createShare,
		"UpdatePortfolioShare":               (*serviceCatalogStore).updateShare,
		"DeletePortfolioShare":               (*serviceCatalogStore).deleteShare,
		"DescribePortfolioShares":            (*serviceCatalogStore).describeShares,
		"DescribePortfolioShareStatus":       (*serviceCatalogStore).shareStatus,
		"AcceptPortfolioShare":               (*serviceCatalogStore).acceptShare,
		"RejectPortfolioShare":               (*serviceCatalogStore).rejectShare,
		"ListAcceptedPortfolioShares":        (*serviceCatalogStore).listAccepted,
		"ListPortfolioAccess":                (*serviceCatalogStore).listPortfolioAccess,
		"ListOrganizationPortfolioAccess":    (*serviceCatalogStore).listOrgAccess,
		"AssociatePrincipalWithPortfolio":    (*serviceCatalogStore).associatePrincipal,
		"DisassociatePrincipalFromPortfolio": (*serviceCatalogStore).disassociatePrincipal,
		"ListPrincipalsForPortfolio":         (*serviceCatalogStore).listPrincipals,
		"AssociateBudgetWithResource":        (*serviceCatalogStore).associateBudget,
		"DisassociateBudgetFromResource":     (*serviceCatalogStore).disassociateBudget,
		"ListBudgetsForResource":             (*serviceCatalogStore).listBudgets,
		"EnableAWSOrganizationsAccess":       (*serviceCatalogStore).enableOrgAccess,
		"DisableAWSOrganizationsAccess":      (*serviceCatalogStore).disableOrgAccess,
		"GetAWSOrganizationsAccessStatus":    (*serviceCatalogStore).orgAccessStatus,
		"CreateTagOption":                    (*serviceCatalogStore).createTagOption,
		"DescribeTagOption":                  (*serviceCatalogStore).describeTagOption,
		"UpdateTagOption":                    (*serviceCatalogStore).updateTagOption,
		"DeleteTagOption":                    (*serviceCatalogStore).deleteTagOption,
		"ListTagOptions":                     (*serviceCatalogStore).listTagOptions,
		"AssociateTagOptionWithResource":     (*serviceCatalogStore).associateTagOption,
		"DisassociateTagOptionFromResource":  (*serviceCatalogStore).disassociateTagOption,
		"ListResourcesForTagOption":          (*serviceCatalogStore).listResourcesForTagOption,
	} {
		svcatHandlers[name] = h
	}
	svcatRegisterProductHandlers()
	svcatRegisterProvisionHandlers()
}

func (st *serviceCatalogStore) portfolio(id string) (*svcatPortfolio, *svcatError) {
	p := st.portfolios[id]
	if p == nil {
		return nil, svcatNotFound("Portfolio %s not found", id)
	}
	return p, nil
}

func (p *svcatPortfolio) detail() map[string]any {
	return map[string]any{"Id": p.id, "ARN": p.arn, "DisplayName": p.name, "Description": p.desc,
		"CreatedTime": svcatEpoch(p.created), "ProviderName": p.provider}
}

func (st *serviceCatalogStore) createPortfolio(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	if id, ok := st.svcatIdempotent("CreatePortfolio", svcatStr(in, "IdempotencyToken")); ok {
		if p := st.portfolios[id]; p != nil {
			return map[string]any{"PortfolioDetail": p.detail(), "Tags": svcatTags(p.tags)}, nil
		}
	}
	if len(st.portfolios) >= svcatMaxPortfolios {
		return nil, svcatErr("LimitExceededException", "an account can have at most %d portfolios", svcatMaxPortfolios)
	}
	id := svcatNewID("port-")
	p := &svcatPortfolio{id: id, arn: svcatPortfolioARN(id), name: svcatStr(in, "DisplayName"), desc: svcatStr(in, "Description"),
		provider: svcatStr(in, "ProviderName"), created: time.Now().UTC(), tags: map[string]string{}, products: map[string]bool{},
		shares: map[string]*svcatShare{}, tagOptions: map[string]bool{}, budgets: map[string]bool{}}
	if err := svcatApplyTags(p.tags, svcatList(in, "Tags"), nil); err != nil {
		return nil, err
	}
	if err := st.rememberToken("CreatePortfolio", svcatStr(in, "IdempotencyToken"), id); err != nil {
		return nil, err
	}
	st.portfolios[id] = p
	return map[string]any{"PortfolioDetail": p.detail(), "Tags": svcatTags(p.tags)}, nil
}

func (st *serviceCatalogStore) describePortfolio(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"PortfolioDetail": p.detail(), "Tags": svcatTags(p.tags),
		"TagOptions": st.tagOptionDetails(p.tagOptions), "Budgets": svcatBudgets(p.budgets)}, nil
}

func (st *serviceCatalogStore) updatePortfolio(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	if err := svcatApplyTags(p.tags, svcatList(in, "AddTags"), svcatList(in, "RemoveTags")); err != nil {
		return nil, err
	}
	if v, ok := in["DisplayName"].(string); ok {
		p.name = v
	}
	if v, ok := in["Description"].(string); ok {
		p.desc = v
	}
	if v, ok := in["ProviderName"].(string); ok {
		p.provider = v
	}
	return map[string]any{"PortfolioDetail": p.detail(), "Tags": svcatTags(p.tags)}, nil
}

func (st *serviceCatalogStore) deletePortfolio(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	switch {
	case len(p.products) > 0:
		return nil, svcatErr("ResourceInUseException", "Portfolio %s still has products associated", p.id)
	case len(p.shares) > 0:
		return nil, svcatErr("ResourceInUseException", "Portfolio %s is still shared", p.id)
	case len(p.principals) > 0:
		return nil, svcatErr("ResourceInUseException", "Portfolio %s still has principals associated", p.id)
	}
	for _, c := range st.constraints {
		if c.portfolio == p.id {
			return nil, svcatErr("ResourceInUseException", "Portfolio %s still has constraints", p.id)
		}
	}
	delete(st.portfolios, p.id)
	delete(st.accepted, p.id)
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) sortedPortfolios(keep func(*svcatPortfolio) bool) []any {
	var ps []*svcatPortfolio
	for _, p := range st.portfolios {
		if keep == nil || keep(p) {
			ps = append(ps, p)
		}
	}
	sort.Slice(ps, func(i, j int) bool {
		if !ps[i].created.Equal(ps[j].created) {
			return ps[i].created.Before(ps[j].created)
		}
		return ps[i].id < ps[j].id
	})
	out := make([]any, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.detail())
	}
	return out
}

func (st *serviceCatalogStore) listPortfolios(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	page, next, err := svcatPage(st.sortedPortfolios(nil), in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"PortfolioDetails": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) listPortfoliosForProduct(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	prod, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	page, next, err := svcatPage(st.sortedPortfolios(func(p *svcatPortfolio) bool { return p.products[prod.id] }), in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"PortfolioDetails": page, "NextPageToken": next}, nil
}

// ---- sharing ----

// shareTarget reads AccountId or OrganizationNode (exactly one).
func (st *serviceCatalogStore) shareTarget(in map[string]any) (typ, value string, err *svcatError) {
	acct, node := svcatStr(in, "AccountId"), svcatMap(in, "OrganizationNode")
	switch {
	case acct != "" && node != nil:
		return "", "", svcatInvalid("Specify either AccountId or OrganizationNode, not both")
	case acct != "":
		if acct == svcatAccount {
			return "", "", svcatInvalid("A portfolio can't be shared with its own account")
		}
		return "ACCOUNT", acct, nil
	case node != nil:
		if st.orgAccess != "ENABLED" {
			return "", "", svcatErr("OperationNotSupportedException", "AWS Organizations access is not enabled for Service Catalog")
		}
		typ, value = svcatStr(node, "Type"), svcatStr(node, "Value")
		if typ == "" || value == "" {
			return "", "", svcatInvalid("OrganizationNode needs a Type and a Value")
		}
		if !svcatOrgNodePattern[typ].MatchString(value) {
			return "", "", svcatInvalid("%s is not a valid %s", value, typ)
		}
		return typ, value, nil
	}
	return "", "", svcatInvalid("Specify AccountId or OrganizationNode")
}

var svcatOrgNodePattern = map[string]*regexp.Regexp{
	"ORGANIZATION":        regexp.MustCompile(`^o-[a-z0-9]{10,32}$`),
	"ORGANIZATIONAL_UNIT": regexp.MustCompile(`^ou-[0-9a-z]{4,32}-[a-z0-9]{8,32}$`),
	"ACCOUNT":             regexp.MustCompile(`^[0-9]{12}$`),
}

func (st *serviceCatalogStore) createShare(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	typ, value, err := st.shareTarget(in)
	if err != nil {
		return nil, err
	}
	if len(p.shares) >= svcatMaxShares {
		return nil, svcatErr("LimitExceededException", "a portfolio can have at most %d shares", svcatMaxShares)
	}
	key := typ + "|" + value
	tagOpts, _ := in["ShareTagOptions"].(bool)
	principals, _ := in["SharePrincipals"].(bool)
	if sh := p.shares[key]; sh != nil {
		sh.shareTagOptions, sh.sharePrincipals = tagOpts, principals
	} else {
		p.shares[key] = &svcatShare{typ: typ, value: value, shareTagOptions: tagOpts, sharePrincipals: principals}
	}
	// AWS documents the token for organization shares; account shares get one
	// too so DescribePortfolioShareStatus works for every share.
	return map[string]any{"PortfolioShareToken": st.newShareToken(p.id, value)}, nil
}

func (st *serviceCatalogStore) newShareToken(portfolio, node string) string {
	tok := svcatNewID("share-")
	if len(st.shareTokens) >= svcatMaxShares*10 {
		for k := range st.shareTokens { // drop an arbitrary old token
			delete(st.shareTokens, k)
			break
		}
	}
	st.shareTokens[tok] = &svcatShareToken{token: tok, portfolio: portfolio, node: node, status: "COMPLETED"}
	return tok
}

func (st *serviceCatalogStore) updateShare(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	typ, value, err := st.shareTarget(in)
	if err != nil {
		return nil, err
	}
	sh := p.shares[typ+"|"+value]
	if sh == nil {
		return nil, svcatNotFound("Portfolio %s is not shared with %s", p.id, value)
	}
	if v, ok := in["ShareTagOptions"].(bool); ok {
		sh.shareTagOptions = v
	}
	if v, ok := in["SharePrincipals"].(bool); ok {
		sh.sharePrincipals = v
	}
	return map[string]any{"PortfolioShareToken": st.newShareToken(p.id, value), "Status": "COMPLETED"}, nil
}

func (st *serviceCatalogStore) deleteShare(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	typ, value, err := st.shareTarget(in)
	if err != nil {
		return nil, err
	}
	if p.shares[typ+"|"+value] == nil {
		return nil, svcatNotFound("Portfolio %s is not shared with %s", p.id, value)
	}
	delete(p.shares, typ+"|"+value)
	return map[string]any{"PortfolioShareToken": st.newShareToken(p.id, value)}, nil
}

func (st *serviceCatalogStore) describeShares(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	want := svcatStr(in, "Type")
	if want == "ORGANIZATION_MEMBER_ACCOUNT" {
		want = "ACCOUNT"
	}
	details := []any{}
	for _, k := range svcatSortedKeys(p.shares) {
		sh := p.shares[k]
		if sh.typ != want {
			continue
		}
		details = append(details, map[string]any{"PrincipalId": sh.value, "Type": svcatStr(in, "Type"), "Accepted": sh.accepted,
			"ShareTagOptions": sh.shareTagOptions, "SharePrincipals": sh.sharePrincipals})
	}
	page, next, err := svcatPage(details, in, 50)
	if err != nil {
		return nil, err
	}
	return map[string]any{"PortfolioShareDetails": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) shareStatus(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	t := st.shareTokens[svcatStr(in, "PortfolioShareToken")]
	if t == nil {
		return nil, svcatNotFound("Portfolio share token %s not found", svcatStr(in, "PortfolioShareToken"))
	}
	return map[string]any{"PortfolioShareToken": t.token, "PortfolioId": t.portfolio, "OrganizationNodeValue": t.node, "Status": t.status,
		"ShareDetails": map[string]any{"SuccessfulShares": []any{t.node}, "ShareErrors": []any{}}}, nil
}

// ponytail: Stackyard has one account, so accepting a share marks one of its
// own portfolios as imported rather than copying a portfolio from another account.
func (st *serviceCatalogStore) acceptShare(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	typ := svcatStr(in, "PortfolioShareType")
	if typ == "" {
		typ = "IMPORTED"
	}
	if typ == "AWS_ORGANIZATIONS" && st.orgAccess != "ENABLED" {
		return nil, svcatErr("OperationNotSupportedException", "AWS Organizations access is not enabled for Service Catalog")
	}
	st.accepted[p.id] = typ
	for _, sh := range p.shares {
		sh.accepted = true
	}
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) rejectShare(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	if _, ok := st.accepted[p.id]; !ok && len(p.shares) == 0 {
		return nil, svcatNotFound("Portfolio %s has no share to reject", p.id)
	}
	delete(st.accepted, p.id)
	for _, sh := range p.shares {
		sh.accepted = false
	}
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) listAccepted(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	want := svcatStr(in, "PortfolioShareType")
	if want == "" {
		want = "IMPORTED"
	}
	page, next, err := svcatPage(st.sortedPortfolios(func(p *svcatPortfolio) bool { return st.accepted[p.id] == want }), in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"PortfolioDetails": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) listPortfolioAccess(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	accounts := []any{}
	for _, k := range svcatSortedKeys(p.shares) {
		if p.shares[k].typ == "ACCOUNT" {
			accounts = append(accounts, p.shares[k].value)
		}
	}
	page, next, err := svcatPage(accounts, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"AccountIds": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) listOrgAccess(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	if st.orgAccess != "ENABLED" {
		return nil, svcatErr("OperationNotSupportedException", "AWS Organizations access is not enabled for Service Catalog")
	}
	want := svcatStr(in, "OrganizationNodeType")
	nodes := []any{}
	for _, k := range svcatSortedKeys(p.shares) {
		if sh := p.shares[k]; sh.typ == want {
			nodes = append(nodes, map[string]any{"Type": sh.typ, "Value": sh.value})
		}
	}
	page, next, err := svcatPage(nodes, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"OrganizationNodes": page, "NextPageToken": next}, nil
}

// ---- principals ----

var svcatPrincipalARN = map[string]*regexp.Regexp{
	"IAM":         regexp.MustCompile(`^arn:aws[a-z-]*:iam::[0-9]{12}:(user|role|group)/[\w+=,.@/-]+$`),
	"IAM_PATTERN": regexp.MustCompile(`^arn:aws[a-z-]*:iam::[0-9]{0,12}:(user|role|group)/[\w+=,.@/*?-]+$`),
}

func (st *serviceCatalogStore) associatePrincipal(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	arn, typ := svcatStr(in, "PrincipalARN"), svcatStr(in, "PrincipalType")
	if !svcatPrincipalARN[typ].MatchString(arn) {
		return nil, svcatInvalid("%s is not a valid %s principal", arn, typ)
	}
	for _, pr := range p.principals {
		if pr.arn == arn {
			return map[string]any{}, nil
		}
	}
	if len(p.principals) >= svcatMaxPrincipals {
		return nil, svcatErr("LimitExceededException", "a portfolio can have at most %d principals", svcatMaxPrincipals)
	}
	p.principals = append(p.principals, svcatPrincipal{arn: arn, typ: typ})
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) disassociatePrincipal(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	arn := svcatStr(in, "PrincipalARN")
	for i, pr := range p.principals {
		if pr.arn == arn {
			p.principals = append(p.principals[:i:i], p.principals[i+1:]...)
			return map[string]any{}, nil
		}
	}
	return nil, svcatNotFound("Principal %s is not associated with portfolio %s", arn, p.id)
}

func (st *serviceCatalogStore) listPrincipals(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, pr := range p.principals {
		items = append(items, map[string]any{"PrincipalARN": pr.arn, "PrincipalType": pr.typ})
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Principals": page, "NextPageToken": next}, nil
}

// ---- budgets ----

// budgetSet returns the budget set of a portfolio or product id.
func (st *serviceCatalogStore) budgetSet(id string) (map[string]bool, *svcatError) {
	if p := st.portfolios[id]; p != nil {
		return p.budgets, nil
	}
	if prod := st.products[id]; prod != nil {
		return prod.budgets, nil
	}
	return nil, svcatNotFound("Resource %s not found", id)
}

func (st *serviceCatalogStore) associateBudget(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	set, err := st.budgetSet(svcatStr(in, "ResourceId"))
	if err != nil {
		return nil, err
	}
	name := svcatStr(in, "BudgetName")
	if set[name] {
		return nil, svcatErr("DuplicateResourceException", "Budget %s is already associated", name)
	}
	if len(set) >= svcatMaxBudgets {
		return nil, svcatErr("LimitExceededException", "a resource can have at most %d budgets", svcatMaxBudgets)
	}
	set[name] = true
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) disassociateBudget(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	set, err := st.budgetSet(svcatStr(in, "ResourceId"))
	if err != nil {
		return nil, err
	}
	name := svcatStr(in, "BudgetName")
	if !set[name] {
		return nil, svcatNotFound("Budget %s is not associated with %s", name, svcatStr(in, "ResourceId"))
	}
	delete(set, name)
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) listBudgets(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	set, err := st.budgetSet(svcatStr(in, "ResourceId"))
	if err != nil {
		return nil, err
	}
	page, next, err := svcatPage(svcatBudgets(set), in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Budgets": page, "NextPageToken": next}, nil
}

func svcatBudgets(set map[string]bool) []any {
	out := []any{}
	for _, b := range svcatSet(set) {
		out = append(out, map[string]any{"BudgetName": b})
	}
	return out
}

// ---- AWS Organizations access ----

func (st *serviceCatalogStore) enableOrgAccess(*Server, map[string]any, *svcatPre) (map[string]any, *svcatError) {
	st.orgAccess = "ENABLED"
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) disableOrgAccess(*Server, map[string]any, *svcatPre) (map[string]any, *svcatError) {
	for _, p := range st.portfolios {
		for _, sh := range p.shares {
			if sh.typ != "ACCOUNT" {
				return nil, svcatErr("ResourceInUseException", "Portfolio %s is still shared with an organization node", p.id)
			}
		}
	}
	st.orgAccess = "DISABLED"
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) orgAccessStatus(*Server, map[string]any, *svcatPre) (map[string]any, *svcatError) {
	return map[string]any{"AccessStatus": st.orgAccess}, nil
}

// ---- tag options ----

func (t *svcatTagOption) detail() map[string]any {
	return map[string]any{"Key": t.key, "Value": t.value, "Active": t.active, "Id": t.id, "Owner": svcatAccount}
}

func (st *serviceCatalogStore) tagOptionDetails(set map[string]bool) []any {
	out := []any{}
	for _, id := range svcatSet(set) {
		if t := st.tagOptions[id]; t != nil {
			out = append(out, t.detail())
		}
	}
	return out
}

// tagOptionSummaries groups tag options by key (DescribeProvisioningParameters).
func (st *serviceCatalogStore) tagOptionSummaries(sets ...map[string]bool) []any {
	byKey := map[string][]any{}
	for _, set := range sets {
		for id := range set {
			if t := st.tagOptions[id]; t != nil && t.active && !svcatContains(byKey[t.key], t.value) {
				byKey[t.key] = append(byKey[t.key], t.value)
			}
		}
	}
	out := []any{}
	for _, k := range svcatSortedKeys(byKey) {
		out = append(out, map[string]any{"Key": k, "Values": byKey[k]})
	}
	return out
}

func (st *serviceCatalogStore) tagOption(id string) (*svcatTagOption, *svcatError) {
	t := st.tagOptions[id]
	if t == nil {
		return nil, svcatNotFound("TagOption %s not found", id)
	}
	return t, nil
}

func (st *serviceCatalogStore) createTagOption(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	key, value := svcatStr(in, "Key"), svcatStr(in, "Value")
	for _, t := range st.tagOptions {
		if t.key == key && t.value == value {
			return nil, svcatErr("DuplicateResourceException", "TagOption %s=%s already exists", key, value)
		}
	}
	if len(st.tagOptions) >= svcatMaxTagOptions {
		return nil, svcatErr("LimitExceededException", "an account can have at most %d TagOptions", svcatMaxTagOptions)
	}
	t := &svcatTagOption{id: svcatNewID("tag-"), key: key, value: value, active: true}
	st.tagOptions[t.id] = t
	return map[string]any{"TagOptionDetail": t.detail()}, nil
}

func (st *serviceCatalogStore) describeTagOption(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	t, err := st.tagOption(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"TagOptionDetail": t.detail()}, nil
}

func (st *serviceCatalogStore) updateTagOption(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	t, err := st.tagOption(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	if v, ok := in["Value"].(string); ok && v != t.value {
		for _, o := range st.tagOptions {
			if o.key == t.key && o.value == v {
				return nil, svcatErr("DuplicateResourceException", "TagOption %s=%s already exists", t.key, v)
			}
		}
		t.value = v
	}
	if v, ok := in["Active"].(bool); ok {
		t.active = v
	}
	return map[string]any{"TagOptionDetail": t.detail()}, nil
}

func (st *serviceCatalogStore) tagOptionInUse(id string) bool {
	for _, p := range st.portfolios {
		if p.tagOptions[id] {
			return true
		}
	}
	for _, prod := range st.products {
		if prod.tagOptions[id] {
			return true
		}
	}
	return false
}

func (st *serviceCatalogStore) deleteTagOption(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	t, err := st.tagOption(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	if st.tagOptionInUse(t.id) {
		return nil, svcatErr("ResourceInUseException", "TagOption %s is still associated with resources", t.id)
	}
	delete(st.tagOptions, t.id)
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) listTagOptions(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	f := svcatMap(in, "Filters")
	var ts []*svcatTagOption
	for _, t := range st.tagOptions {
		if k := svcatStr(f, "Key"); k != "" && t.key != k {
			continue
		}
		if v := svcatStr(f, "Value"); v != "" && t.value != v {
			continue
		}
		if a, ok := f["Active"].(bool); ok && t.active != a {
			continue
		}
		ts = append(ts, t)
	}
	sort.Slice(ts, func(i, j int) bool {
		if ts[i].key != ts[j].key {
			return ts[i].key < ts[j].key
		}
		return ts[i].value < ts[j].value
	})
	items := make([]any, 0, len(ts))
	for _, t := range ts {
		items = append(items, t.detail())
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"TagOptionDetails": page, "PageToken": next}, nil
}

// tagOptionSet returns the tag option set of a portfolio or product id.
func (st *serviceCatalogStore) tagOptionSet(id string) (map[string]bool, *svcatError) {
	if p := st.portfolios[id]; p != nil {
		return p.tagOptions, nil
	}
	if prod := st.products[id]; prod != nil {
		return prod.tagOptions, nil
	}
	return nil, svcatNotFound("Resource %s not found", id)
}

func (st *serviceCatalogStore) associateTagOption(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	t, err := st.tagOption(svcatStr(in, "TagOptionId"))
	if err != nil {
		return nil, err
	}
	set, err := st.tagOptionSet(svcatStr(in, "ResourceId"))
	if err != nil {
		return nil, err
	}
	if !t.active {
		return nil, svcatInvalid("TagOption %s is not active", t.id)
	}
	if set[t.id] {
		return nil, svcatErr("DuplicateResourceException", "TagOption %s is already associated", t.id)
	}
	for id := range set { // one value per key on a resource
		if o := st.tagOptions[id]; o != nil && o.key == t.key {
			return nil, svcatErr("DuplicateResourceException", "A TagOption with key %s is already associated", t.key)
		}
	}
	set[t.id] = true
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) disassociateTagOption(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	set, err := st.tagOptionSet(svcatStr(in, "ResourceId"))
	if err != nil {
		return nil, err
	}
	id := svcatStr(in, "TagOptionId")
	if !set[id] {
		return nil, svcatNotFound("TagOption %s is not associated with %s", id, svcatStr(in, "ResourceId"))
	}
	delete(set, id)
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) listResourcesForTagOption(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	t, err := st.tagOption(svcatStr(in, "TagOptionId"))
	if err != nil {
		return nil, err
	}
	want := strings.ToLower(svcatStr(in, "ResourceType"))
	items := []any{}
	if want == "" || want == "portfolio" {
		for _, d := range st.sortedPortfolios(func(p *svcatPortfolio) bool { return p.tagOptions[t.id] }) {
			m := d.(map[string]any)
			items = append(items, map[string]any{"Id": m["Id"], "ARN": m["ARN"], "Name": m["DisplayName"], "Description": m["Description"], "CreatedTime": m["CreatedTime"]})
		}
	}
	if want == "" || want == "product" {
		for _, prod := range st.sortedProducts(func(p *svcatProduct) bool { return p.tagOptions[t.id] }) {
			items = append(items, map[string]any{"Id": prod.id, "ARN": prod.arn, "Name": prod.name, "Description": prod.desc, "CreatedTime": svcatEpoch(prod.created)})
		}
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ResourceDetails": page, "PageToken": next}, nil
}
