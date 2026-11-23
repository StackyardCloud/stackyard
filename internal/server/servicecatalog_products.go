package server

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

func svcatRegisterProductHandlers() {
	for name, h := range map[string]svcatHandler{
		"CreateProduct":                    (*serviceCatalogStore).createProduct,
		"DescribeProduct":                  (*serviceCatalogStore).describeProduct,
		"DescribeProductAsAdmin":           (*serviceCatalogStore).describeProductAsAdmin,
		"DescribeProductView":              (*serviceCatalogStore).describeProductView,
		"UpdateProduct":                    (*serviceCatalogStore).updateProduct,
		"DeleteProduct":                    (*serviceCatalogStore).deleteProduct,
		"SearchProducts":                   (*serviceCatalogStore).searchProducts,
		"SearchProductsAsAdmin":            (*serviceCatalogStore).searchProductsAsAdmin,
		"AssociateProductWithPortfolio":    (*serviceCatalogStore).associateProduct,
		"DisassociateProductFromPortfolio": (*serviceCatalogStore).disassociateProduct,
		"ListLaunchPaths":                  (*serviceCatalogStore).listLaunchPaths,
		"CreateProvisioningArtifact":       (*serviceCatalogStore).createArtifact,
		"DescribeProvisioningArtifact":     (*serviceCatalogStore).describeArtifact,
		"UpdateProvisioningArtifact":       (*serviceCatalogStore).updateArtifact,
		"DeleteProvisioningArtifact":       (*serviceCatalogStore).deleteArtifact,
		"ListProvisioningArtifacts":        (*serviceCatalogStore).listArtifacts,
		"DescribeProvisioningParameters":   (*serviceCatalogStore).describeProvisioningParameters,
		"CreateConstraint":                 (*serviceCatalogStore).createConstraint,
		"DescribeConstraint":               (*serviceCatalogStore).describeConstraint,
		"UpdateConstraint":                 (*serviceCatalogStore).updateConstraint,
		"DeleteConstraint":                 (*serviceCatalogStore).deleteConstraint,
		"ListConstraintsForPortfolio":      (*serviceCatalogStore).listConstraints,
		"CopyProduct":                      (*serviceCatalogStore).copyProduct,
		"DescribeCopyProductStatus":        (*serviceCatalogStore).copyStatus,
	} {
		svcatHandlers[name] = h
	}
}

func (st *serviceCatalogStore) product(id string) (*svcatProduct, *svcatError) {
	p := st.products[id]
	if p == nil {
		return nil, svcatNotFound("Product %s not found", id)
	}
	return p, nil
}

// productByIDOrName resolves the Id / Name pair many describe calls take.
func (st *serviceCatalogStore) productByIDOrName(id, name string) (*svcatProduct, *svcatError) {
	if id != "" {
		return st.product(id)
	}
	if name == "" {
		return nil, svcatInvalid("Specify the product Id or Name")
	}
	for _, p := range st.products {
		if p.name == name {
			return p, nil
		}
	}
	return nil, svcatNotFound("Product %s not found", name)
}

func (p *svcatProduct) summary(st *serviceCatalogStore) map[string]any {
	return map[string]any{"Id": p.viewID, "ProductId": p.id, "Name": p.name, "Owner": p.owner, "ShortDescription": p.desc,
		"Type": p.typ, "Distributor": p.distributor, "HasDefaultPath": len(st.launchPortfolios(p)) == 1,
		"SupportEmail": p.supportEmail, "SupportDescription": p.supportDesc, "SupportUrl": p.supportURL}
}

func (p *svcatProduct) viewDetail(st *serviceCatalogStore) map[string]any {
	d := map[string]any{"ProductViewSummary": p.summary(st), "Status": "AVAILABLE", "ProductARN": p.arn, "CreatedTime": svcatEpoch(p.created)}
	if p.source != nil {
		d["SourceConnection"] = map[string]any{"Type": "CODESTAR", "ConnectionParameters": p.source["ConnectionParameters"],
			"LastSync": map[string]any{"LastSyncStatus": "SUCCEEDED", "LastSyncTime": svcatEpoch(p.created), "LastSuccessfulSyncTime": svcatEpoch(p.created)}}
	}
	return d
}

func (a *svcatArtifact) detail() map[string]any {
	return map[string]any{"Id": a.id, "Name": a.name, "Description": a.desc, "Type": a.typ, "CreatedTime": svcatEpoch(a.created),
		"Active": a.active, "Guidance": a.guidance}
}

func (a *svcatArtifact) infoMap() map[string]any {
	out := map[string]any{}
	for k, v := range a.info {
		out[k] = v
	}
	return out
}

// artifactType maps a product type to its default provisioning artifact type.
func svcatArtifactType(productType, given string) (string, *svcatError) {
	if given != "" {
		switch {
		case productType == "CLOUD_FORMATION_TEMPLATE" && given != "CLOUD_FORMATION_TEMPLATE",
			strings.HasPrefix(productType, "TERRAFORM") && given != productType,
			productType == "EXTERNAL" && given != "EXTERNAL":
			return "", svcatInvalid("Provisioning artifact type %s doesn't match product type %s", given, productType)
		}
		return given, nil
	}
	if productType == "MARKETPLACE" {
		return "MARKETPLACE_AMI", nil
	}
	return productType, nil
}

// newArtifact builds a provisioning artifact from ProvisioningArtifactProperties.
func svcatNewArtifact(productType string, props map[string]any, pre *svcatPre) (*svcatArtifact, *svcatError) {
	typ, err := svcatArtifactType(productType, svcatStr(props, "Type"))
	if err != nil {
		return nil, err
	}
	info := svcatMap(props, "Info")
	if svcatStr(info, "LoadTemplateFromURL") == "" && svcatStr(info, "ImportFromPhysicalId") == "" {
		return nil, svcatInvalid("Info must contain LoadTemplateFromURL or ImportFromPhysicalId")
	}
	if pre.templateErr != nil {
		return nil, pre.templateErr
	}
	skip, _ := props["DisableTemplateValidation"].(bool)
	tmpl, err := svcatParseTemplate(typ, pre.template, skip)
	if err != nil {
		return nil, err
	}
	a := &svcatArtifact{id: svcatNewID("pa-"), name: svcatStr(props, "Name"), desc: svcatStr(props, "Description"), typ: typ,
		guidance: "DEFAULT", active: true, created: time.Now().UTC(), info: map[string]string{}, tmpl: tmpl}
	for k, v := range info {
		if s, ok := v.(string); ok {
			a.info[k] = s
		}
	}
	if a.name == "" {
		a.name = "v1"
	}
	return a, nil
}

func (st *serviceCatalogStore) createProduct(_ *Server, in map[string]any, pre *svcatPre) (map[string]any, *svcatError) {
	if id, ok := st.svcatIdempotent("CreateProduct", svcatStr(in, "IdempotencyToken")); ok {
		if p := st.products[id]; p != nil {
			return st.productCreated(p), nil
		}
	}
	if len(st.products) >= svcatMaxProducts {
		return nil, svcatErr("LimitExceededException", "an account can have at most %d products", svcatMaxProducts)
	}
	typ := svcatStr(in, "ProductType")
	if typ == "MARKETPLACE" {
		return nil, svcatInvalid("MARKETPLACE products are created by AWS Marketplace, not CreateProduct")
	}
	id := svcatNewID("prod-")
	p := &svcatProduct{id: id, viewID: svcatNewID("prodview-"), arn: svcatProductARN(id), name: svcatStr(in, "Name"), owner: svcatStr(in, "Owner"),
		desc: svcatStr(in, "Description"), distributor: svcatStr(in, "Distributor"), supportDesc: svcatStr(in, "SupportDescription"),
		supportEmail: svcatStr(in, "SupportEmail"), supportURL: svcatStr(in, "SupportUrl"), typ: typ, created: time.Now().UTC(),
		tags: map[string]string{}, tagOptions: map[string]bool{}, budgets: map[string]bool{}, source: svcatMap(in, "SourceConnection")}
	if err := svcatApplyTags(p.tags, svcatList(in, "Tags"), nil); err != nil {
		return nil, err
	}
	props := svcatMap(in, "ProvisioningArtifactParameters")
	if props == nil && p.source == nil {
		return nil, svcatInvalid("ProvisioningArtifactParameters or SourceConnection is required")
	}
	if props != nil {
		a, err := svcatNewArtifact(typ, props, pre)
		if err != nil {
			return nil, err
		}
		p.artifacts = append(p.artifacts, a)
	}
	if err := st.rememberToken("CreateProduct", svcatStr(in, "IdempotencyToken"), id); err != nil {
		return nil, err
	}
	st.products[id] = p
	st.views[p.viewID] = id
	return st.productCreated(p), nil
}

func (st *serviceCatalogStore) productCreated(p *svcatProduct) map[string]any {
	out := map[string]any{"ProductViewDetail": p.viewDetail(st), "Tags": svcatTags(p.tags)}
	if len(p.artifacts) > 0 {
		out["ProvisioningArtifactDetail"] = p.artifacts[0].detail()
	}
	return out
}

func (p *svcatProduct) activeArtifacts() []any {
	out := []any{}
	for _, a := range p.artifacts {
		if a.active {
			out = append(out, map[string]any{"Id": a.id, "Name": a.name, "Description": a.desc, "CreatedTime": svcatEpoch(a.created), "Guidance": a.guidance})
		}
	}
	return out
}

func (st *serviceCatalogStore) describeProduct(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.productByIDOrName(svcatStr(in, "Id"), svcatStr(in, "Name"))
	if err != nil {
		return nil, err
	}
	paths := []any{}
	for _, port := range st.launchPortfolios(p) {
		paths = append(paths, map[string]any{"Id": svcatPathID(port.id), "Name": port.name})
	}
	if len(paths) == 0 {
		return nil, svcatNotFound("Product %s is not available to you: it isn't in any portfolio", p.id)
	}
	return map[string]any{"ProductViewSummary": p.summary(st), "ProvisioningArtifacts": p.activeArtifacts(),
		"Budgets": svcatBudgets(p.budgets), "LaunchPaths": paths}, nil
}

func (st *serviceCatalogStore) describeProductAsAdmin(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.productByIDOrName(svcatStr(in, "Id"), svcatStr(in, "Name"))
	if err != nil {
		return nil, err
	}
	if src := svcatStr(in, "SourcePortfolioId"); src != "" {
		if port := st.portfolios[src]; port == nil || !port.products[p.id] {
			return nil, svcatNotFound("Product %s is not in portfolio %s", p.id, src)
		}
	}
	summaries := []any{}
	for _, a := range p.artifacts {
		summaries = append(summaries, map[string]any{"Id": a.id, "Name": a.name, "Description": a.desc, "CreatedTime": svcatEpoch(a.created),
			"ProvisioningArtifactMetadata": a.infoMap()})
	}
	return map[string]any{"ProductViewDetail": p.viewDetail(st), "ProvisioningArtifactSummaries": summaries, "Tags": svcatTags(p.tags),
		"TagOptions": st.tagOptionDetails(p.tagOptions), "Budgets": svcatBudgets(p.budgets)}, nil
}

func (st *serviceCatalogStore) describeProductView(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p := st.products[st.views[svcatStr(in, "Id")]]
	if p == nil {
		return nil, svcatNotFound("Product view %s not found", svcatStr(in, "Id"))
	}
	return map[string]any{"ProductViewSummary": p.summary(st), "ProvisioningArtifacts": p.activeArtifacts()}, nil
}

func (st *serviceCatalogStore) updateProduct(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	if err := svcatApplyTags(p.tags, svcatList(in, "AddTags"), svcatList(in, "RemoveTags")); err != nil {
		return nil, err
	}
	for key, field := range map[string]*string{"Name": &p.name, "Owner": &p.owner, "Description": &p.desc, "Distributor": &p.distributor,
		"SupportDescription": &p.supportDesc, "SupportEmail": &p.supportEmail, "SupportUrl": &p.supportURL} {
		if v, ok := in[key].(string); ok {
			*field = v
		}
	}
	if src := svcatMap(in, "SourceConnection"); src != nil {
		p.source = src
	}
	return map[string]any{"ProductViewDetail": p.viewDetail(st), "Tags": svcatTags(p.tags)}, nil
}

func (st *serviceCatalogStore) deleteProduct(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	if len(st.launchPortfolios(p)) > 0 {
		return nil, svcatErr("ResourceInUseException", "Product %s is still associated with a portfolio", p.id)
	}
	for _, pp := range st.provisioned {
		if pp.productID == p.id {
			return nil, svcatErr("ResourceInUseException", "Product %s has provisioned products", p.id)
		}
	}
	for _, act := range st.actions {
		for k := range act.assoc {
			if k[0] == p.id {
				delete(act.assoc, k)
			}
		}
	}
	delete(st.products, p.id)
	delete(st.views, p.viewID)
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) sortedProducts(keep func(*svcatProduct) bool) []*svcatProduct {
	var out []*svcatProduct
	for _, p := range st.products {
		if keep == nil || keep(p) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].created.Equal(out[j].created) {
			return out[i].created.Before(out[j].created)
		}
		return out[i].id < out[j].id
	})
	return out
}

// productFilter applies SearchProducts filters and sort order.
func svcatFilterProducts(ps []*svcatProduct, in map[string]any) []*svcatProduct {
	filters := svcatMap(in, "Filters")
	var out []*svcatProduct
	for _, p := range ps {
		ok := true
		for key, vals := range filters {
			list, _ := vals.([]any)
			match := len(list) == 0
			for _, v := range list {
				s := strings.ToLower(svcatAnyString(v))
				switch key {
				case "FullTextSearch":
					hay := strings.ToLower(p.name + " " + p.desc + " " + p.owner + " " + p.distributor)
					match = match || strings.Contains(hay, s)
				case "Owner":
					match = match || strings.EqualFold(p.owner, s)
				case "ProductType":
					match = match || strings.EqualFold(p.typ, s)
				case "SourceProductId":
					match = match || strings.EqualFold(p.id, s)
				}
			}
			ok = ok && match
		}
		if ok {
			out = append(out, p)
		}
	}
	desc := svcatStr(in, "SortOrder") == "DESCENDING"
	switch svcatStr(in, "SortBy") {
	case "Title":
		sort.SliceStable(out, func(i, j int) bool { return (out[i].name < out[j].name) != desc })
	case "VersionCount":
		sort.SliceStable(out, func(i, j int) bool { return (len(out[i].artifacts) < len(out[j].artifacts)) != desc })
	case "CreationDate":
		sort.SliceStable(out, func(i, j int) bool { return out[i].created.Before(out[j].created) != desc })
	}
	return out
}

func svcatAnyString(v any) string { s, _ := v.(string); return s }

func (st *serviceCatalogStore) searchProducts(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	ps := svcatFilterProducts(st.sortedProducts(func(p *svcatProduct) bool { return len(st.launchPortfolios(p)) > 0 }), in)
	items := make([]any, 0, len(ps))
	owners, types := map[string]int{}, map[string]int{}
	for _, p := range ps {
		items = append(items, p.summary(st))
		owners[p.owner]++
		types[p.typ]++
	}
	agg := func(counts map[string]int) []any {
		out := []any{}
		for _, k := range svcatSortedKeys(counts) {
			out = append(out, map[string]any{"Value": k, "ApproximateCount": counts[k]})
		}
		return out
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ProductViewSummaries": page, "NextPageToken": next,
		"ProductViewAggregations": map[string]any{"Owner": agg(owners), "ProductType": agg(types)}}, nil
}

func (st *serviceCatalogStore) searchProductsAsAdmin(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	var keep func(*svcatProduct) bool
	if id := svcatStr(in, "PortfolioId"); id != "" {
		port, err := st.portfolio(id)
		if err != nil {
			return nil, err
		}
		keep = func(p *svcatProduct) bool { return port.products[p.id] }
	}
	ps := svcatFilterProducts(st.sortedProducts(keep), in)
	items := make([]any, 0, len(ps))
	for _, p := range ps {
		items = append(items, p.viewDetail(st))
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ProductViewDetails": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) associateProduct(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	port, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	if src := svcatStr(in, "SourcePortfolioId"); src != "" {
		if sp := st.portfolios[src]; sp == nil || !sp.products[p.id] {
			return nil, svcatNotFound("Product %s is not in source portfolio %s", p.id, src)
		}
	}
	port.products[p.id] = true
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) disassociateProduct(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	port, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	if !port.products[p.id] {
		return nil, svcatNotFound("Product %s is not in portfolio %s", p.id, port.id)
	}
	for _, c := range st.constraints {
		if c.portfolio == port.id && c.product == p.id {
			return nil, svcatErr("ResourceInUseException", "Remove the constraints on product %s in portfolio %s first", p.id, port.id)
		}
	}
	delete(port.products, p.id)
	return map[string]any{}, nil
}

// launchPortfolios are the portfolios a product can be launched from.
func (st *serviceCatalogStore) launchPortfolios(p *svcatProduct) []*svcatPortfolio {
	var out []*svcatPortfolio
	for _, port := range st.portfolios {
		if port.products[p.id] {
			out = append(out, port)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

// Launch path ids name the portfolio a product is launched through.
func svcatPathID(portfolioID string) string {
	return "lpv3-" + strings.TrimPrefix(portfolioID, "port-")
}

// resolvePath picks the launch path for a product: the one named, or the only one.
func (st *serviceCatalogStore) resolvePath(p *svcatProduct, pathID, pathName string) (*svcatPortfolio, *svcatError) {
	ports := st.launchPortfolios(p)
	for _, port := range ports {
		if (pathID != "" && svcatPathID(port.id) == pathID) || (pathID == "" && pathName != "" && port.name == pathName) {
			return port, nil
		}
	}
	switch {
	case pathID != "" || pathName != "":
		return nil, svcatNotFound("Launch path %s%s not found for product %s", pathID, pathName, p.id)
	case len(ports) == 0:
		return nil, svcatNotFound("Product %s has no launch path: add it to a portfolio", p.id)
	case len(ports) > 1:
		return nil, svcatInvalid("Product %s has several launch paths; specify PathId", p.id)
	}
	return ports[0], nil
}

func (st *serviceCatalogStore) constraintSummaries(portfolio, product string) []any {
	out := []any{}
	for _, c := range st.sortedConstraints(portfolio, product) {
		out = append(out, map[string]any{"Type": c.typ, "Description": c.desc})
	}
	return out
}

func (st *serviceCatalogStore) listLaunchPaths(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, port := range st.launchPortfolios(p) {
		items = append(items, map[string]any{"Id": svcatPathID(port.id), "Name": port.name, "Tags": svcatTags(port.tags),
			"ConstraintSummaries": st.constraintSummaries(port.id, p.id)})
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"LaunchPathSummaries": page, "NextPageToken": next}, nil
}

// ---- provisioning artifacts ----

func (p *svcatProduct) artifact(id, name string) (*svcatArtifact, *svcatError) {
	for _, a := range p.artifacts {
		if (id != "" && a.id == id) || (id == "" && name != "" && a.name == name) {
			return a, nil
		}
	}
	if id == "" && name == "" {
		return nil, svcatInvalid("Specify the provisioning artifact Id or Name")
	}
	return nil, svcatNotFound("Provisioning artifact %s%s not found for product %s", id, name, p.id)
}

func (st *serviceCatalogStore) createArtifact(_ *Server, in map[string]any, pre *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	if id, ok := st.svcatIdempotent("CreateProvisioningArtifact", svcatStr(in, "IdempotencyToken")); ok {
		if a, err := p.artifact(id, ""); err == nil {
			return map[string]any{"ProvisioningArtifactDetail": a.detail(), "Info": a.infoMap(), "Status": "AVAILABLE"}, nil
		}
	}
	if len(p.artifacts) >= svcatMaxArtifacts {
		return nil, svcatErr("LimitExceededException", "a product can have at most %d provisioning artifacts", svcatMaxArtifacts)
	}
	props := svcatMap(in, "Parameters")
	a, err := svcatNewArtifact(p.typ, props, pre)
	if err != nil {
		return nil, err
	}
	if svcatStr(props, "Name") == "" {
		a.name = "v" + strconv.Itoa(len(p.artifacts)+1)
	}
	for _, o := range p.artifacts {
		if o.name == a.name {
			return nil, svcatErr("DuplicateResourceException", "Provisioning artifact %s already exists", a.name)
		}
	}
	if err := st.rememberToken("CreateProvisioningArtifact", svcatStr(in, "IdempotencyToken"), a.id); err != nil {
		return nil, err
	}
	p.artifacts = append(p.artifacts, a)
	return map[string]any{"ProvisioningArtifactDetail": a.detail(), "Info": a.infoMap(), "Status": "AVAILABLE"}, nil
}

func (st *serviceCatalogStore) describeArtifact(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.productByIDOrName(svcatStr(in, "ProductId"), svcatStr(in, "ProductName"))
	if err != nil {
		return nil, err
	}
	a, err := p.artifact(svcatStr(in, "ProvisioningArtifactId"), svcatStr(in, "ProvisioningArtifactName"))
	if err != nil {
		return nil, err
	}
	out := map[string]any{"ProvisioningArtifactDetail": a.detail(), "Status": "AVAILABLE"}
	if v, _ := in["Verbose"].(bool); v {
		out["Info"] = a.infoMap()
	}
	if v, _ := in["IncludeProvisioningArtifactParameters"].(bool); v {
		out["ProvisioningArtifactParameters"] = a.tmpl.paramShapes()
	}
	return out, nil
}

func (st *serviceCatalogStore) updateArtifact(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	a, err := p.artifact(svcatStr(in, "ProvisioningArtifactId"), "")
	if err != nil {
		return nil, err
	}
	if v, ok := in["Name"].(string); ok && v != a.name {
		for _, o := range p.artifacts {
			if o.name == v {
				return nil, svcatErr("DuplicateResourceException", "Provisioning artifact %s already exists", v)
			}
		}
		a.name = v
	}
	if v, ok := in["Description"].(string); ok {
		a.desc = v
	}
	if v, ok := in["Active"].(bool); ok {
		a.active = v
	}
	if v, ok := in["Guidance"].(string); ok {
		a.guidance = v
	}
	return map[string]any{"ProvisioningArtifactDetail": a.detail(), "Info": a.infoMap(), "Status": "AVAILABLE"}, nil
}

func (st *serviceCatalogStore) deleteArtifact(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	a, err := p.artifact(svcatStr(in, "ProvisioningArtifactId"), "")
	if err != nil {
		return nil, err
	}
	if len(p.artifacts) == 1 {
		return nil, svcatInvalid("A product must keep at least one provisioning artifact")
	}
	for _, pp := range st.provisioned {
		if pp.artifactID == a.id {
			return nil, svcatErr("ResourceInUseException", "Provisioning artifact %s is used by provisioned product %s", a.id, pp.id)
		}
	}
	for i, o := range p.artifacts {
		if o == a {
			p.artifacts = append(p.artifacts[:i:i], p.artifacts[i+1:]...)
			break
		}
	}
	for _, act := range st.actions {
		delete(act.assoc, [2]string{p.id, a.id})
	}
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) listArtifacts(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, a := range p.artifacts {
		items = append(items, a.detail())
	}
	return map[string]any{"ProvisioningArtifactDetails": items}, nil
}

func (st *serviceCatalogStore) describeProvisioningParameters(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.productByIDOrName(svcatStr(in, "ProductId"), svcatStr(in, "ProductName"))
	if err != nil {
		return nil, err
	}
	a, err := p.artifact(svcatStr(in, "ProvisioningArtifactId"), svcatStr(in, "ProvisioningArtifactName"))
	if err != nil {
		return nil, err
	}
	port, err := st.resolvePath(p, svcatStr(in, "PathId"), svcatStr(in, "PathName"))
	if err != nil {
		return nil, err
	}
	outputs := []any{}
	for _, o := range a.tmpl.outputs {
		outputs = append(outputs, map[string]any{"Key": o.key, "Description": o.desc})
	}
	usage := []any{}
	for _, c := range st.sortedConstraints(port.id, p.id) {
		usage = append(usage, map[string]any{"Type": strings.ToLower(c.typ), "Value": c.params})
	}
	return map[string]any{"ProvisioningArtifactParameters": a.tmpl.paramShapes(), "ConstraintSummaries": st.constraintSummaries(port.id, p.id),
		"UsageInstructions": usage, "TagOptions": st.tagOptionSummaries(p.tagOptions, port.tagOptions),
		"ProvisioningArtifactOutputs": outputs, "ProvisioningArtifactOutputKeys": outputs}, nil
}

// ---- constraints ----

func (c *svcatConstraint) detail() map[string]any {
	return map[string]any{"ConstraintId": c.id, "Type": c.typ, "Description": c.desc, "Owner": svcatAccount, "ProductId": c.product, "PortfolioId": c.portfolio}
}

func (st *serviceCatalogStore) sortedConstraints(portfolio, product string) []*svcatConstraint {
	var out []*svcatConstraint
	for _, c := range st.constraints {
		if (portfolio == "" || c.portfolio == portfolio) && (product == "" || c.product == product) {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

// svcatCheckConstraint validates Parameters for a constraint type.
func svcatCheckConstraint(typ, params string) *svcatError {
	var p map[string]any
	if err := json.Unmarshal([]byte(params), &p); err != nil {
		return svcatInvalid("Parameters must be a JSON object")
	}
	has := func(k string) bool { _, ok := p[k]; return ok }
	switch typ {
	case "LAUNCH":
		role, local := svcatStr(p, "RoleArn"), svcatStr(p, "LocalRoleName")
		if (role == "") == (local == "") {
			return svcatInvalid("A LAUNCH constraint needs exactly one of RoleArn or LocalRoleName")
		}
		if role != "" && !svcatPrincipalARN["IAM"].MatchString(role) {
			return svcatInvalid("RoleArn %s is not an IAM role ARN", role)
		}
	case "NOTIFICATION":
		arns, _ := p["NotificationArns"].([]any)
		if len(arns) == 0 {
			return svcatInvalid("A NOTIFICATION constraint needs NotificationArns")
		}
		for _, a := range arns {
			if s, _ := a.(string); !strings.HasPrefix(s, "arn:aws:sns:") {
				return svcatInvalid("%v is not an SNS topic ARN", a)
			}
		}
	case "RESOURCE_UPDATE":
		v := svcatStr(p, "TagUpdatesOnProvisionedProduct")
		if v != "ALLOWED" && v != "NOT_ALLOWED" {
			return svcatInvalid("TagUpdatesOnProvisionedProduct must be ALLOWED or NOT_ALLOWED")
		}
	case "STACKSET":
		if svcatStr(p, "Version") != "Version1" || !has("AccountList") || !has("RegionList") || svcatStr(p, "AdminRole") == "" || svcatStr(p, "ExecutionRole") == "" {
			return svcatInvalid("A STACKSET constraint needs Version1, AccountList, RegionList, AdminRole and ExecutionRole")
		}
	case "TEMPLATE":
		if _, ok := p["Rules"].(map[string]any); !ok {
			return svcatInvalid("A TEMPLATE constraint needs a Rules object")
		}
	default:
		return svcatInvalid("Constraint type %s is not one of LAUNCH, NOTIFICATION, RESOURCE_UPDATE, STACKSET or TEMPLATE", typ)
	}
	return nil
}

func (st *serviceCatalogStore) constraintOut(c *svcatConstraint) map[string]any {
	return map[string]any{"ConstraintDetail": c.detail(), "ConstraintParameters": c.params, "Status": "AVAILABLE"}
}

func (st *serviceCatalogStore) constraint(id string) (*svcatConstraint, *svcatError) {
	c := st.constraints[id]
	if c == nil {
		return nil, svcatNotFound("Constraint %s not found", id)
	}
	return c, nil
}

func (st *serviceCatalogStore) createConstraint(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	if id, ok := st.svcatIdempotent("CreateConstraint", svcatStr(in, "IdempotencyToken")); ok {
		if c := st.constraints[id]; c != nil {
			return st.constraintOut(c), nil
		}
	}
	port, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	if !port.products[p.id] {
		return nil, svcatInvalid("Product %s is not in portfolio %s", p.id, port.id)
	}
	typ, params := svcatStr(in, "Type"), svcatStr(in, "Parameters")
	if err := svcatCheckConstraint(typ, params); err != nil {
		return nil, err
	}
	for _, c := range st.sortedConstraints(port.id, p.id) {
		if c.typ == typ && (typ == "LAUNCH" || typ == "STACKSET" || typ == "RESOURCE_UPDATE") {
			return nil, svcatErr("DuplicateResourceException", "Product %s already has a %s constraint in portfolio %s", p.id, typ, port.id)
		}
	}
	if len(st.constraints) >= svcatMaxConstraints {
		return nil, svcatErr("LimitExceededException", "an account can have at most %d constraints", svcatMaxConstraints)
	}
	c := &svcatConstraint{id: svcatNewID("cons-"), typ: typ, desc: svcatStr(in, "Description"), params: params, portfolio: port.id, product: p.id}
	if err := st.rememberToken("CreateConstraint", svcatStr(in, "IdempotencyToken"), c.id); err != nil {
		return nil, err
	}
	st.constraints[c.id] = c
	return st.constraintOut(c), nil
}

func (st *serviceCatalogStore) describeConstraint(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	c, err := st.constraint(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	return st.constraintOut(c), nil
}

func (st *serviceCatalogStore) updateConstraint(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	c, err := st.constraint(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	if v, ok := in["Parameters"].(string); ok {
		if err := svcatCheckConstraint(c.typ, v); err != nil {
			return nil, err
		}
		c.params = v
	}
	if v, ok := in["Description"].(string); ok {
		c.desc = v
	}
	return st.constraintOut(c), nil
}

func (st *serviceCatalogStore) deleteConstraint(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	c, err := st.constraint(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	delete(st.constraints, c.id)
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) listConstraints(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	port, err := st.portfolio(svcatStr(in, "PortfolioId"))
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, c := range st.sortedConstraints(port.id, svcatStr(in, "ProductId")) {
		items = append(items, c.detail())
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ConstraintDetails": page, "NextPageToken": next}, nil
}

// ---- copy ----

func (st *serviceCatalogStore) copyProduct(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	if tok, ok := st.svcatIdempotent("CopyProduct", svcatStr(in, "IdempotencyToken")); ok {
		return map[string]any{"CopyProductToken": tok}, nil
	}
	if len(st.copies) >= svcatMaxTokens {
		return nil, svcatErr("LimitExceededException", "too many product copies are held")
	}
	arn := svcatStr(in, "SourceProductArn")
	var src *svcatProduct
	for _, p := range st.products {
		if p.arn == arn {
			src = p
		}
	}
	if src == nil {
		return nil, svcatNotFound("Source product %s not found", arn)
	}
	// Which artifacts to copy: the listed ids, or all of them.
	var artifacts []*svcatArtifact
	for _, m := range svcatList(in, "SourceProvisioningArtifactIdentifiers") {
		id := svcatStr(svcatAsMap(m), "Id")
		a, err := src.artifact(id, "")
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, a)
	}
	if artifacts == nil {
		artifacts = src.artifacts
	}
	var target *svcatProduct
	if id := svcatStr(in, "TargetProductId"); id != "" {
		t, err := st.product(id)
		if err != nil {
			return nil, err
		}
		if len(t.artifacts)+len(artifacts) > svcatMaxArtifacts {
			return nil, svcatErr("LimitExceededException", "a product can have at most %d provisioning artifacts", svcatMaxArtifacts)
		}
		target = t
	} else {
		if len(st.products) >= svcatMaxProducts {
			return nil, svcatErr("LimitExceededException", "an account can have at most %d products", svcatMaxProducts)
		}
		id := svcatNewID("prod-")
		name := svcatStr(in, "TargetProductName")
		if name == "" {
			name = src.name
		}
		target = &svcatProduct{id: id, viewID: svcatNewID("prodview-"), arn: svcatProductARN(id), name: name, owner: src.owner, desc: src.desc,
			distributor: src.distributor, supportDesc: src.supportDesc, supportEmail: src.supportEmail, supportURL: src.supportURL, typ: src.typ,
			created: time.Now().UTC(), tags: map[string]string{}, tagOptions: map[string]bool{}, budgets: map[string]bool{}}
		for _, o := range svcatList(in, "CopyOptions") {
			if o == "CopyTags" {
				for k, v := range src.tags {
					target.tags[k] = v
				}
			}
		}
		st.products[id] = target
		st.views[target.viewID] = id
	}
	for _, a := range artifacts {
		cp := *a
		cp.id, cp.created = svcatNewID("pa-"), time.Now().UTC()
		cp.info = map[string]string{}
		for k, v := range a.info {
			cp.info[k] = v
		}
		target.artifacts = append(target.artifacts, &cp)
	}
	tok := svcatNewID("copyproduct-")
	if err := st.rememberToken("CopyProduct", svcatStr(in, "IdempotencyToken"), tok); err != nil {
		return nil, err
	}
	// ponytail: the copy happens at once; DescribeCopyProductStatus reports SUCCEEDED.
	st.copies[tok] = &svcatCopy{token: tok, status: "SUCCEEDED", target: target.id}
	return map[string]any{"CopyProductToken": tok}, nil
}

func (st *serviceCatalogStore) copyStatus(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	c := st.copies[svcatStr(in, "CopyProductToken")]
	if c == nil {
		return nil, svcatNotFound("Copy product token %s not found", svcatStr(in, "CopyProductToken"))
	}
	return map[string]any{"CopyProductStatus": c.status, "TargetProductId": c.target, "StatusDetail": c.detail}, nil
}

func svcatAsMap(v any) map[string]any { m, _ := v.(map[string]any); return m }
